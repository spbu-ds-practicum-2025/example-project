package config

import (
	"os"
)

// Config holds all configuration for the Analytics Service
type Config struct {
	GRPCPort   string
	ClickHouse ClickHouseConfig
	RabbitMQ   RabbitMQConfig
}

// ClickHouseConfig holds ClickHouse connection configuration
type ClickHouseConfig struct {
	Host     string
	Database string
	User     string
	Password string
}

// RabbitMQConfig holds RabbitMQ connection configuration
type RabbitMQConfig struct {
	URL        string
	Queue      string
	Exchange   string
	RoutingKey string
}

// Load loads configuration from environment variables with default values
func Load() *Config {
	return &Config{
		GRPCPort: getEnv("GRPC_PORT", "50053"),
		ClickHouse: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost:9000"),
			Database: getEnv("CLICKHOUSE_DB", "analytics"),
			User:     getEnv("CLICKHOUSE_USER", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
		},
		RabbitMQ: RabbitMQConfig{
			URL:        getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			Queue:      getEnv("RABBITMQ_QUEUE", "analytics.transfer.completed"),
			Exchange:   getEnv("RABBITMQ_EXCHANGE", "bank.operations"),
			RoutingKey: getEnv("RABBITMQ_ROUTING_KEY", "bank.operations.transfer.completed"),
		},
	}
}

// getEnv retrieves an environment variable or returns a default value if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
