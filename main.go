package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type TelegramRequest struct {
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}

func sendMessage(token, chatID, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]string{
		"chat_id": chatID,
		"text":    message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(responseBody))
	}

	return nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	botToken := os.Getenv("TELEGRAM_TOKEN")
	accessToken := os.Getenv("ACCESS_TOKEN")

	if botToken == "" || accessToken == "" {
		log.Fatal("TELEGRAM_TOKEN or ACCESS_TOKEN is not set")
	}

	http.HandleFunc("/service/proxy/telegram", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clientToken := r.Header.Get("X-Access-Token")
		if clientToken != accessToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req TelegramRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.ChatID == "" || req.Message == "" {
			http.Error(w, "Missing chat_id or message", http.StatusBadRequest)
			return
		}

		if err := sendMessage(botToken, req.ChatID, req.Message); err != nil {
			log.Println("Failed to send message:", err)
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
