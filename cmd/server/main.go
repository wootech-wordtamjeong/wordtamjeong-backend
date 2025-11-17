package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/joho/godotenv"
	"github.com/yourusername/kkomantl/internal/handlers"
	"github.com/yourusername/kkomantl/internal/services"
	"github.com/yourusername/kkomantl/pkg/bedrock"
)


func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(getEnv("AWS_REGION", "ap-northeast-2")),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Initialize embedding service
	modelID := getEnv("AWS_BEDROCK_MODEL", "amazon.titan-embed-text-v2:0")
	embeddingService := bedrock.NewEmbeddingService(cfg, modelID)

	// Initialize quiz service
	quizService := services.NewQuizService(embeddingService)

	// Initialize handlers
	handler := handlers.NewHandler(quizService)

	// Setup routes
	http.HandleFunc("/health", handler.CORSMiddleware(handler.HealthHandler))
	http.HandleFunc("/api/guess", handler.CORSMiddleware(handler.GuessHandler))
	http.HandleFunc("/api/status", handler.CORSMiddleware(handler.StatusHandler))
	http.HandleFunc("/admin/rotate", handler.CORSMiddleware(handler.AdminRotateHandler))

	// Start server
	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	log.Printf("Starting server on %s", addr)
	log.Printf("Using model: %s", modelID)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
