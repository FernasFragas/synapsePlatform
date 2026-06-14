//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestEndToEnd_SummaryFlow(t *testing.T) {
	apiBase := os.Getenv("E2E_API_URL")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	ollamaBase := os.Getenv("E2E_OLLAMA_URL")
	if ollamaBase == "" {
		ollamaBase = "http://localhost:11435"
	}

	// 0. Wait for Ollama model to be available
	waitForOllamaModel(t, ollamaBase)

	// 1. Ingest sample events via Kafka
	seedEvents(t)

	// 2. Wait for ingestion — poll /v1/events instead of fixed sleep
	token := generateTestToken(t)
	waitForEvents(t, apiBase, token)

	// 3. Call /v1/summary
	req, err := http.NewRequest(http.MethodGet, apiBase+"/v1/summary?domain=energy&since=2024-01-01T00:00:00Z", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 65 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("summary failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var report struct {
		Domain    string `json:"domain"`
		Model     string `json:"model"`
		Content   string `json:"content"`
		CreatedAt string `json:"created_at"`
	}
	require.NoError(t, json.Unmarshal(body, &report))

	require.Equal(t, "energy", report.Domain)
	require.NotEmpty(t, report.Model)
	require.NotEmpty(t, report.Content)
	require.NotEmpty(t, report.CreatedAt)

	t.Logf("Summary: %s (model: %s)", report.Content, report.Model)

	// 4. Call again — should hit cache
	resp2, err := client.Do(req)
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	body2, _ := io.ReadAll(resp2.Body)
	var report2 struct {
		CreatedAt string `json:"created_at"`
	}
	require.NoError(t, json.Unmarshal(body2, &report2))
	require.Equal(t, report.CreatedAt, report2.CreatedAt, "cache should return same timestamp")
}

func waitForOllamaModel(t *testing.T, base string) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}

	targetModel := "mistral:7b"

	for i := 0; i < 60; i++ {
		resp, err := client.Get(base + "/api/tags")
		if err != nil {
			t.Logf("ollama poll attempt %d: connection error: %v", i, err)
			time.Sleep(2 * time.Second)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Logf("ollama poll attempt %d: status=%d", i, resp.StatusCode)
			time.Sleep(2 * time.Second)
			continue
		}
		if readErr != nil {
			t.Logf("ollama poll attempt %d: read error: %v", i, readErr)
			time.Sleep(2 * time.Second)
			continue
		}

		var out struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}

		if err := json.Unmarshal(body, &out); err != nil {
			t.Logf("ollama poll attempt %d: json decode error: %v", i, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if len(out.Models) == 0 {
			t.Logf("ollama poll attempt %d: no models yet", i)
			time.Sleep(2 * time.Second)
			continue
		}

		// List all models for debugging
		t.Logf("ollama poll attempt %d: found %d models", i, len(out.Models))
		for _, m := range out.Models {
			t.Logf("  - model: %q (len=%d)", m.Name, len(m.Name))
			if m.Name == targetModel {
				t.Logf("Ollama model %q ready after %d attempts (%d seconds)", targetModel, i+1, i*2)
				return
			}
		}

		time.Sleep(2 * time.Second)
	}
	t.Fatalf("ollama model %q not available after 2 minutes", targetModel)
}

func seedEvents(t *testing.T) {
	t.Helper()

	// Verify jq exists
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skipf("jq not installed, skipping Kafka seed: %v", err)
	}

	// Verify Kafka container is running
	kafkaContainer := detectKafkaContainer(t)
	t.Logf("Using Kafka container: %s", kafkaContainer)

	// Send events directly without make, so we control the container name
	projectRoot := "../.."
	files, err := filepath.Glob(filepath.Join(projectRoot, "test", "*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no test JSON files found in test/")

	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)

		// Minify JSON with jq
		minified, err := minifyJSON(data)
		require.NoError(t, err, "failed to minify %s", file)

		cmd := exec.Command("docker", "exec", "-i", kafkaContainer,
			"kafka-console-producer",
			"--broker-list", "localhost:9092",
			"--topic", "ingestion.raw",
		)
		cmd.Stdin = bytes.NewReader(minified)

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "failed to send %s: %s", file, string(out))
		t.Logf("Sent %s (%d bytes)", filepath.Base(file), len(minified))
	}
}

func detectKafkaContainer(t *testing.T) string {
	t.Helper()
	// Try common naming patterns
	candidates := []string{
		"synapseplatform-kafka-1",
		"synapsePlatform-kafka-1",
		"synapse-platform-kafka-1",
	}
	for _, name := range candidates {
		cmd := exec.Command("docker", "ps", "-q", "-f", "name="+name)
		out, err := cmd.Output()
		if err == nil && len(bytes.TrimSpace(out)) > 0 {
			return name
		}
	}
	t.Fatal("no Kafka container found; expected one of: " + strings.Join(candidates, ", "))
	return ""
}

func minifyJSON(data []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func waitForEvents(t *testing.T, apiBase, token string) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, apiBase+"/v1/events", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	for i := 0; i < 180; i++ { // 180 seconds max
		resp, err := client.Do(req.Clone(req.Context()))
		if err != nil {
			t.Logf("poll %d: request error: %v", i, err)
			time.Sleep(1 * time.Second)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Logf("poll %d: status=%d body=%s", i, resp.StatusCode, string(bodyBytes))
			time.Sleep(1 * time.Second)
			continue
		}

		var body struct {
			Data []any `json:"data"`
		}
		if json.Unmarshal(bodyBytes, &body) == nil && len(body.Data) >= 3 {
			t.Logf("Events appeared after %d seconds: %d items", i, len(body.Data))
			return
		}

		t.Logf("poll %d: 200 OK but %d events", i, len(body.Data))
		time.Sleep(1 * time.Second)
	}
	t.Fatal("events never appeared in API after 180 seconds")
}

func generateTestToken(t *testing.T) string {
	t.Helper()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-256-bit-secret-replace-me!!"
	}

	claims := jwt.MapClaims{
		"iss":       "https://auth.example.com",
		"aud":       "synapse-platform-api",
		"sub":       "e2e-test-user",
		"client_id": "e2e-client",
		"scope":     "read:events",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	return ss
}
