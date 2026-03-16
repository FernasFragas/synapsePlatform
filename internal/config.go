package internal

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Auth     AuthConfig     `yaml:"auth"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Database DatabaseConfig `yaml:"database"`
	Tracing  TracingConfig  `yaml:"tracing"`
	Log      LogConfig      `yaml:"log"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.Auth.JWT.Secret = secret
	}
	return cfg, nil
}

type ServerConfig struct {
	Address   string          `yaml:"address"`
	Timeouts  ServerTimeouts  `yaml:"timeouts"`
	Shutdown  ShutdownConfig  `yaml:"shutdown"`
	CORS      CORSConfig      `yaml:"cors"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type ServerTimeouts struct {
	ReadHeader time.Duration `yaml:"read_header"`
	Read       time.Duration `yaml:"read"`
	Write      time.Duration `yaml:"write"`
	Idle       time.Duration `yaml:"idle"`
}

type ShutdownConfig struct {
	Timeout     time.Duration `yaml:"timeout"`
	GracePeriod time.Duration `yaml:"grace_period"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
}

type AuthConfig struct {
	JWT JWTConfig `yaml:"jwt"`
}

type JWTConfig struct {
	Issuer   string `yaml:"issuer"`
	Audience string `yaml:"audience"`
	Secret   string `yaml:"secret"`
}

type KafkaConfig struct {
	Brokers   []string `yaml:"brokers"`
	GroupID   string   `yaml:"group_id"`
	Topics    []string `yaml:"topics"`
	DLQTopics string   `yaml:"dlq_topic"`
	MinBytes  int      `yaml:"min_bytes"`
	MaxBytes  int      `yaml:"max_bytes"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LogConfig struct {
	RedactKeys    []string `yaml:"redact_keys"`
	MaxValueBytes int      `yaml:"max_value_bytes"`
}

type TracingConfig struct {
	Endpoint string `yaml:"endpoint"`
}
