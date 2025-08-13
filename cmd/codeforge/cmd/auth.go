package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/api"
    "github.com/spf13/cobra"
)

var (
	authPort      int
	authForeground bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Start local auth server and obtain a token",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure global app was initialized by root PersistentPreRunE
		if codeforgeApp == nil {
			return fmt.Errorf("app not initialized")
		}

		// Create and start API server with integrated app
		server := api.NewServerWithApp(codeforgeApp.Config, codeforgeApp)
		if server == nil {
			return fmt.Errorf("failed to create API server")
		}

		// Start server in background
		done := make(chan error, 1)
		go func() { done <- server.Start(authPort) }()

		// Wait for server health
		baseURL := fmt.Sprintf("http://localhost:%d", authPort)
		healthURL := fmt.Sprintf("%s/api/v1/health", baseURL)
		if err := waitForHealthy(healthURL, 5*time.Second); err != nil {
			return fmt.Errorf("server did not become healthy: %w", err)
		}

		// Request a token (localhost-only)
		loginURL := fmt.Sprintf("%s/api/v1/auth", baseURL)
		reqBody := bytes.NewBufferString(`{"device_name":"codeforge-cli"}`)
		resp, err := http.Post(loginURL, "application/json", reqBody)
		if err != nil {
			return fmt.Errorf("login request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("login failed: %s", string(b))
		}
		var payload struct {
			Token     string `json:"token"`
			SessionID string `json:"session_id"`
			Message   string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return fmt.Errorf("failed to parse login response: %w", err)
		}

		fmt.Printf("Auth server running on %s\n", baseURL)
		fmt.Printf("Token: %s\n", payload.Token)
		fmt.Printf("Session: %s\n", payload.SessionID)
		fmt.Printf("Use the token in Authorization: Bearer <token>\n")

		if !authForeground {
			// Non-foreground: give server a moment then exit (leaves server running in this process)
			return nil
		}

		// Foreground: wait for interrupt
		fmt.Println("Press Ctrl+C to stop the auth server...")
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		select {
		case <-sig:
			fmt.Println("\nStopping auth server...")
			return nil
		case err := <-done:
			return err
		}
	},
}

func init() {
	authCmd.Flags().IntVar(&authPort, "port", 47000, "Port to run the auth server on")
	authCmd.Flags().BoolVar(&authForeground, "foreground", true, "Keep the server running in the foreground")
	rootCmd.AddCommand(authCmd)
}

func waitForHealthy(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	client := &http.Client{Timeout: 1 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := client.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}
