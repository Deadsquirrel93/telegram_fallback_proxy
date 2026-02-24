# 📬 Telegram Fallback Proxy Bot Service

[🇷🇺 Русский](Readme.md) | [🇬🇧 English](Readme_en.md)

A proxy service for sending messages to Telegram via your bot.  
Receives HTTP requests and forwards texts to the specified `chat_id`.

**Why use this proxy?**  
1. **Reliable Message Delivery (Fallback):** This project was originally designed as a "safeguard route" for instances when main application servers encountered network instability or constant timeouts while trying to reach `api.telegram.org` (e.g., due to regional network restrictions or ISP routing issues). By deploying this lightweight proxy on a node with a stable internet connection, you can guarantee the reliable delivery of all your Telegram notifications.
2. **Token Security:** It serves as an excellent architectural pattern for hiding your actual Telegram bot token from client applications (frontend, mobile apps) or internal microservices. You centralize the notification logic and interact with the proxy using a specific internal `ACCESS_TOKEN`.

## 🌟 Features

- 🪶 **Super-lightweight Docker image** (~27 MB) thanks to multi-stage builds.
- ⏱️ **Built-in HTTP server timeouts**, protecting against DDoS (Slowloris attacks).
- 🛡️ **Memory protection** with proper limits on incoming JSON payload sizes (up to 1 MB).
- 🔄 **Re-utilised HTTP client** with timeouts for outgoing Telegram API calls, preventing goroutine leaks.
- ⚙️ **Flexible configuration**: bot endpoint and access tokens are easily configurable via environment variables.

---

## ⚙️ Environment Variables

Create an `.env` file with the following setup:

```env
TELEGRAM_TOKEN=your_telegram_bot_token
ACCESS_TOKEN=your_api_access_token
PROXY_ENDPOINT=/service/proxy/telegram
```

- `TELEGRAM_TOKEN` — your Telegram bot token (from BotFather).
- `ACCESS_TOKEN` — an API access token acting as a secret password (passed in request headers).
- `PROXY_ENDPOINT` — custom routing URI, defaults to `/service/proxy/telegram`.

---

## 🚀 Deployment

### 1. Build Docker image

```bash
docker build -t telegram-proxy .
```

### 2. Run the container

```bash
docker run -p 8080:8080 --env-file .env -d telegram-proxy
```

---

## 🔗 Usage

```bash
curl -X POST http://localhost:8080/service/proxy/telegram \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: your_api_access_token" \
  -d '{"chat_id": "123456789", "message": "Message body"}'
```

### Request Payload

| Field    | Type    | Description                   |
|----------|---------|-------------------------------|
| chat_id  | string  | Destination Chat / User ID    |
| message  | string  | The text content to send      |

---

## ✅ API Responses

- `204 No content` — message was sent successfully
- `401 Unauthorized` — `X-Access-Token` was missing or invalid
- `400 Bad Request` — missing `chat_id` or `message`
- `500 Internal Server Error` — error interacting with Telegram API

---

## 📦 `.env` Example

```env
TELEGRAM_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
ACCESS_TOKEN=my-secret-token
PROXY_ENDPOINT=/my/custom/endpoint
```

---

## 🛡️ Security Advice

- If you map ports directly to a public subnet, considering limiting API access via IP whitelisting.
