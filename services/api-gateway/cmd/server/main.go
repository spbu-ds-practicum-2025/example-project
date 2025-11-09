package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/clients"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/handlers"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/server"
)

func main() {
	// Get configuration from environment variables
	bankServiceAddr := getEnv("BANK_SERVICE_ADDR", "localhost:50051")
	analyticsServiceAddr := getEnv("ANALYTICS_SERVICE_ADDR", "localhost:50052")
	port := getEnv("PORT", "8080")

	// Create bank service client
	bankClient, err := clients.NewBankClient(bankServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create bank client: %v", err)
	}
	defer bankClient.Close()

	// Create analytics service client
	analyticsClient, err := clients.NewAnalyticsClient(analyticsServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create analytics client: %v", err)
	}
	defer analyticsClient.Close()

	// Create handler
	handler := handlers.NewHandler(bankClient, analyticsClient)

	// Create HTTP server with generated router
	httpHandler := server.Handler(handler)

	// Start server
	addr := ":" + port
	log.Printf("API Gateway starting on %s, connecting to Bank Service at %s and Analytics Service at %s",
		addr, bankServiceAddr, analyticsServiceAddr)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: httpHandler,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	_ = context.Background()
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
