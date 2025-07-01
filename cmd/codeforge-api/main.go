package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/entrepeneur4lyf/codeforge/internal/api"
	"github.com/entrepeneur4lyf/codeforge/internal/app"
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
		// Set debug mode in configuration
		cfg.Debug = true
		log.Println("Debug mode enabled")
	}

	// Initialize CodeForge application with all systems
	appConfig := &app.AppConfig{
		ConfigPath:        *configPath,
		WorkspaceRoot:     ".",
		EnablePermissions: true,
		EnableContextMgmt: true,
		Debug:             *debug,
	}

	ctx := context.Background()
	codeforgeApp, err := app.NewApp(ctx, appConfig)
	if err != nil {
		log.Fatalf("Failed to initialize CodeForge app: %v", err)
	}
	defer codeforgeApp.Close()

	// Create API server with integrated app
	server := api.NewServerWithApp(cfg, codeforgeApp)
	if server == nil {
		log.Fatalf("Failed to create API server")
	}

	fmt.Printf("Starting CodeForge API Server\n")
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

	log.Println("All services initialized successfully")
	return nil
}
