//go:generate mockgen -source=$GOFILE -destination=../utilstest/mocksgen/health/mocked_health.go
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Probe interface {
	Name() string
	Check(ctx context.Context) error
}

type Checker struct {
	probes  []Probe
	timeout time.Duration
}

func NewChecker(timeout time.Duration, probes ...Probe) *Checker {
	return &Checker{
		probes:  probes,
		timeout: timeout,
	}
}

type Result struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func (c *Checker) CheckAll(ctx context.Context) (int, Result) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	checks := make(map[string]string, len(c.probes))
	allOK := true

	for _, p := range c.probes {
		err := p.Check(ctx)
		if err != nil {
			checks[p.Name()] = err.Error()
			allOK = false
		} else {
			checks[p.Name()] = "ok"
		}
	}

	status := http.StatusOK
	statusText := "healthy"
	if !allOK {
		status = http.StatusServiceUnavailable
		statusText = "unhealthy"
	}

	return status, Result{Status: statusText, Checks: checks}
}

func (c *Checker) ReadyzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, result := c.CheckAll(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func LivezHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Result{Status: "alive", Checks: map[string]string{}})
	}
}
