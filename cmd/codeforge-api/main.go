package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/entrepeneur4lyf/codeforge/internal/api"
	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/embeddings"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

func main() {
	// Parse command line flags
	var (
		port       = flag.Int("port", 47000, "Port to run the API server on")
		configPath = flag.String("config", "", "Path to configuration file")
		debug      = flag.Bool("debug", false, "Enable debug mode")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath, false)
	if err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
		cfg = &config.Config{} // Use empty config
	}

	if *debug {
		// TODO: Set debug mode when config supports it
		log.Println("Debug mode enabled")
	}

	// Initialize services
	if err := initializeServices(cfg); err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Create and start API server
	server := api.NewServer(cfg)

	fmt.Printf("🚀 Starting CodeForge API Server\n")
	fmt.Printf("📡 Server: http://localhost:%d\n", *port)
	fmt.Printf("🔗 Health: http://localhost:%d/api/v1/health\n", *port)
	fmt.Printf("📊 Metrics: http://localhost:%d/api/v1/events/metrics\n", *port)
	fmt.Printf("🔌 WebSocket: ws://localhost:%d/api/v1/chat/ws/{sessionId}\n", *port)

	if err := server.Start(*port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// initializeServices initializes required services
func initializeServices(cfg *config.Config) error {
	// Initialize vector database
	if err := vectordb.Initialize(cfg); err != nil {
		return fmt.Errorf("failed to initialize vector database: %w", err)
	}

	// Initialize embeddings
	if err := embeddings.Initialize(cfg); err != nil {
		return fmt.Errorf("failed to initialize embeddings: %w", err)
	}

	log.Println("✅ All services initialized successfully")
	return nil
}
