package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(*testing.T, *Config)
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.GRPCPort != "50053" {
					t.Errorf("expected GRPCPort to be 50053, got %s", cfg.GRPCPort)
				}
				if cfg.ClickHouse.Host != "localhost:9000" {
					t.Errorf("expected ClickHouse host to be localhost:9000, got %s", cfg.ClickHouse.Host)
				}
				if cfg.ClickHouse.Database != "analytics" {
					t.Errorf("expected ClickHouse database to be analytics, got %s", cfg.ClickHouse.Database)
				}
				if cfg.RabbitMQ.URL != "amqp://guest:guest@localhost:5672/" {
					t.Errorf("expected RabbitMQ URL to be amqp://guest:guest@localhost:5672/, got %s", cfg.RabbitMQ.URL)
				}
			},
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"GRPC_PORT":         "8080",
				"CLICKHOUSE_HOST":   "clickhouse.prod:9000",
				"CLICKHOUSE_DB":     "analytics_prod",
				"CLICKHOUSE_USER":   "admin",
				"CLICKHOUSE_PASSWORD": "secret",
				"RABBITMQ_URL":      "amqp://user:pass@rabbitmq:5672/",
				"RABBITMQ_QUEUE":    "custom.queue",
				"RABBITMQ_EXCHANGE": "custom.exchange",
				"RABBITMQ_ROUTING_KEY": "custom.key",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.GRPCPort != "8080" {
					t.Errorf("expected GRPCPort to be 8080, got %s", cfg.GRPCPort)
				}
				if cfg.ClickHouse.Host != "clickhouse.prod:9000" {
					t.Errorf("expected ClickHouse host to be clickhouse.prod:9000, got %s", cfg.ClickHouse.Host)
				}
				if cfg.ClickHouse.Database != "analytics_prod" {
					t.Errorf("expected ClickHouse database to be analytics_prod, got %s", cfg.ClickHouse.Database)
				}
				if cfg.ClickHouse.User != "admin" {
					t.Errorf("expected ClickHouse user to be admin, got %s", cfg.ClickHouse.User)
				}
				if cfg.ClickHouse.Password != "secret" {
					t.Errorf("expected ClickHouse password to be secret, got %s", cfg.ClickHouse.Password)
				}
				if cfg.RabbitMQ.URL != "amqp://user:pass@rabbitmq:5672/" {
					t.Errorf("expected RabbitMQ URL to be amqp://user:pass@rabbitmq:5672/, got %s", cfg.RabbitMQ.URL)
				}
				if cfg.RabbitMQ.Queue != "custom.queue" {
					t.Errorf("expected RabbitMQ queue to be custom.queue, got %s", cfg.RabbitMQ.Queue)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearEnv()

			// Load configuration
			cfg := Load()

			// Validate
			tt.validate(t, cfg)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "returns env value when set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "returns empty string when env is empty",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv(tt.key)

			// Set env if provided
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			// Test getEnv
			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// clearEnv clears all test environment variables
func clearEnv() {
	envVars := []string{
		"GRPC_PORT",
		"CLICKHOUSE_HOST",
		"CLICKHOUSE_DB",
		"CLICKHOUSE_USER",
		"CLICKHOUSE_PASSWORD",
		"RABBITMQ_URL",
		"RABBITMQ_QUEUE",
		"RABBITMQ_EXCHANGE",
		"RABBITMQ_ROUTING_KEY",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}
}
