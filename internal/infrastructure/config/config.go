package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App          AppConfig
	Backend      BackendConfig
	Observability ObservabilityConfig
	Activities   ActivitiesConfig
}

type AppConfig struct {
	Name    string
	Port    int
	Timeout time.Duration
}

type BackendConfig struct {
	Type          string // "sqlite" or "memory"
	SQLiteFile    string
	MaxConnection int
}

type ObservabilityConfig struct {
	LogLevel       string
	LogFormat      string // "json" or "text"
	MetricsEnabled bool
	MetricsPort    int
	TracingEnabled bool
	ZipkinEndpoint string
}

type ActivitiesConfig struct {
	RetryMaxAttempts    int
	RetryBackoffMs      int
	TimeoutSeconds      int
	CircuitBreakerThreshold float64
	CircuitBreakerTimeout   time.Duration
}

// DefaultConfig returns configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "task-orchestrator",
			Port:    8080,
			Timeout: 30 * time.Second,
		},
		Backend: BackendConfig{
			Type:          "sqlite",
			SQLiteFile:    "data/orchestration.db",
			MaxConnection: 25,
		},
		Observability: ObservabilityConfig{
			LogLevel:       "info",
			LogFormat:      "json",
			MetricsEnabled: true,
			MetricsPort:    9090,
			TracingEnabled: false,
			ZipkinEndpoint: "http://localhost:9411/api/v2/spans",
		},
		Activities: ActivitiesConfig{
			RetryMaxAttempts:        3,
			RetryBackoffMs:          100,
			TimeoutSeconds:          30,
			CircuitBreakerThreshold: 0.5,
			CircuitBreakerTimeout:   10 * time.Second,
		},
	}
}

// LoadConfig loads configuration from YAML file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, err
		}
		if err := viper.Unmarshal(cfg); err != nil {
			return nil, err
		}
	}

	// Environment variable overrides
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	if backend := os.Getenv("APP_BACKEND_TYPE"); backend != "" {
		cfg.Backend.Type = backend
	}
	if sqliteFile := os.Getenv("APP_BACKEND_SQLITE_FILE"); sqliteFile != "" {
		cfg.Backend.SQLiteFile = sqliteFile
	}
	if logLevel := os.Getenv("APP_LOG_LEVEL"); logLevel != "" {
		cfg.Observability.LogLevel = logLevel
	}
	if tracingEnabled := os.Getenv("APP_TRACING_ENABLED"); tracingEnabled != "" {
		cfg.Observability.TracingEnabled = tracingEnabled == "true"
	}
	if zipkinEndpoint := os.Getenv("APP_ZIPKIN_ENDPOINT"); zipkinEndpoint != "" {
		cfg.Observability.ZipkinEndpoint = zipkinEndpoint
	}

	return cfg, nil
}
