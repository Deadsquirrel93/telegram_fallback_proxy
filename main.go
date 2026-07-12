// Package main implements a simple HTTP proxy service that forwards
// incoming JSON requests as text messages to a Telegram bot.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

// TelegramRequest represents the expected JSON payload from the client.
type TelegramRequest struct {
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}

// telegramPayload represents the outgoing JSON structure required by Telegram API.
type telegramPayload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

var (
	// defaultClient is reused across requests. It defines a timeout to prevent Goroutine leaks
	// and hanging if the Telegram API server is slow or unresponsive.
	defaultClient = &http.Client{
		Timeout: 10 * time.Second,
	}
)

const (
	defaultProxyEndpoint            = "/service/proxy/telegram"
	defaultHealthcheckEndpoint      = "/healthz"
	defaultPort                     = "8080"
	defaultShutdownTimeoutSeconds   = 15
)

func normalizeListenAddr(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		port = defaultPort
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}

func authorizeRequest(r *http.Request, accessToken string) bool {
	return r.Header.Get("X-Access-Token") == accessToken
}

// sendMessage sends a standard text message to the specified Telegram chat.
// It uses the provided bot token, encodes the data using JSON, and returns an error
// wrapped with context if the API call fails.
func sendMessage(token, chatID, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload := telegramPayload{
		ChatID: chatID,
		Text:   message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Pre-execute network request
	resp, err := defaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read a limited amount of data to avoid resource exhaustion in case of unexpectedly large error answers
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// checkTelegram verifies that the Telegram Bot API is reachable and the bot
// token is valid by calling the lightweight getMe method. It returns an error
// wrapped with context if the API is unreachable or responds with a non-OK status.
// The provided context bounds the request duration so the healthcheck never
// hangs longer than the caller allows.
func checkTelegram(ctx context.Context, token string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read a limited amount of data to avoid resource exhaustion in case of unexpectedly large error answers
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

func main() {
	// We ignore the error from godotenv.Load().
	// This is critical for Docker / CI environments where .env files do not exist,
	// and environment variables are passed directly to the container.
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, continuing with system environment variables.")
	}

	botToken := os.Getenv("TELEGRAM_TOKEN")
	accessToken := os.Getenv("ACCESS_TOKEN")
	proxyEndpoint := os.Getenv("PROXY_ENDPOINT")
	healthcheckEndpoint := os.Getenv("HEALTHCHECK_ENDPOINT")
	listenAddr := normalizeListenAddr(os.Getenv("PORT"))

	if botToken == "" || accessToken == "" {
		log.Fatal("Error: TELEGRAM_TOKEN and ACCESS_TOKEN must be provided.")
	}
	if proxyEndpoint == "" {
		proxyEndpoint = defaultProxyEndpoint
	}
	if healthcheckEndpoint == "" {
		healthcheckEndpoint = defaultHealthcheckEndpoint
	}

	// Define handler for HTTP POST mapping
	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !authorizeRequest(r, accessToken) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Prevent huge payloads via LimitReader (e.g. 1 MB limit)
		// This protects the service from reading indefinitely large requests.
		limitReader := io.LimitReader(r.Body, 1024*1024)
		var req TelegramRequest
		if err := json.NewDecoder(limitReader).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.ChatID == "" || req.Message == "" {
			http.Error(w, "Missing chat_id or message fields", http.StatusBadRequest)
			return
		}

		if err := sendMessage(botToken, req.ChatID, req.Message); err != nil {
			log.Printf("Failed to send message to [%s]: %v", req.ChatID, err)
			http.Error(w, "Failed to send message via Telegram", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}

	healthcheckHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !authorizeRequest(r, accessToken) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify Telegram API reachability with a bounded timeout so the
		// healthcheck never hangs longer than the server's WriteTimeout.
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp := map[string]string{
			"status":   "ok",
			"telegram": "ok",
		}
		statusCode := http.StatusOK

		if err := checkTelegram(ctx, botToken); err != nil {
			log.Printf("Healthcheck: Telegram API unavailable: %v", err)
			resp["status"] = "degraded"
			resp["telegram"] = "unavailable"
			// Signal unhealthy state to orchestrators / monitors via status code.
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to write healthcheck response: %v", err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc(proxyEndpoint, proxyHandler)
	mux.HandleFunc(healthcheckEndpoint, healthcheckHandler)

	// Use custom http.Server instead of http.ListenAndServe
	// This ensures Read and Write timeouts are set to prevent DDoS (Slowloris attacks)
	// and dangling goroutines.
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	shutdownTimeout := defaultShutdownTimeoutSeconds
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			shutdownTimeout = parsed
		} else {
			log.Printf("Warning: invalid SHUTDOWN_TIMEOUT value %q, using default %ds", v, defaultShutdownTimeoutSeconds)
		}
	}

	// Run server in a separate goroutine so main can block on OS signal.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting Telegram proxy server on %s", listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Block until SIGINT or SIGTERM is received, or the server fails to start.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatal("Server failed to start:", err)
	case sig := <-quit:
		log.Printf("Received signal %s, shutting down gracefully (timeout: %ds)...", sig, shutdownTimeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server stopped.")
}
