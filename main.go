// Package main implements a simple HTTP proxy service that forwards
// incoming JSON requests as text messages to a Telegram bot.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

	if botToken == "" || accessToken == "" {
		log.Fatal("Error: TELEGRAM_TOKEN and ACCESS_TOKEN must be provided.")
	}
	if proxyEndpoint == "" {
		proxyEndpoint = "/service/proxy/telegram"
	}

	// Define handler for HTTP POST mapping
	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clientToken := r.Header.Get("X-Access-Token")
		if clientToken != accessToken {
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

	mux := http.NewServeMux()
	mux.HandleFunc(proxyEndpoint, proxyHandler)

	// Use custom http.Server instead of http.ListenAndServe
	// This ensures Read and Write timeouts are set to prevent DDoS (Slowloris attacks)
	// and dangling goroutines.
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("Starting Telegram proxy server on port :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server shutdown ungracefully:", err)
	}
}
