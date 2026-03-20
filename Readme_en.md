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
- ❤️ **Protected healthcheck** that always requires `X-Access-Token`.
- ⚙️ **Flexible configuration**: port, endpoints, and access tokens are easily configurable via environment variables.

---

## ⚙️ Environment Variables

The service can load configuration from `.env` or from system environment variables. Only `TELEGRAM_TOKEN` and `ACCESS_TOKEN` are strictly required.

```env
TELEGRAM_TOKEN=your_telegram_bot_token
ACCESS_TOKEN=your_api_access_token
PORT=8080
PROXY_ENDPOINT=/service/proxy/telegram
HEALTHCHECK_ENDPOINT=/healthz
```

- `TELEGRAM_TOKEN` — your Telegram bot token (from BotFather).
- `ACCESS_TOKEN` — an API access token acting as a secret password (passed in request headers).
- `PORT` — the HTTP server port inside the container/process, defaults to `8080`.
- `PROXY_ENDPOINT` — custom routing URI, defaults to `/service/proxy/telegram`.
- `HEALTHCHECK_ENDPOINT` — healthcheck URI, defaults to `/healthz`.

---

## 🚀 Deployment

### 1. Build Docker image

```bash
docker build -t telegram-proxy .
```

### 2. Run the container

```bash
docker run -d \
  --name telegram-proxy \
  --restart unless-stopped \
  -p 8080:8080 \
  --env-file .env \
  telegram-proxy
```

`--restart unless-stopped` makes Docker restart the container automatically after a crash or after the Docker daemon restarts.

If you change `PORT`, expose the same port on both sides:

```bash
docker run -d \
  --name telegram-proxy \
  --restart unless-stopped \
  -p 9090:9090 \
  --env-file .env \
  -e PORT=9090 \
  telegram-proxy
```

---

## 🔗 Usage

### Send a message

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

## ❤️ Healthcheck

The healthcheck does not call Telegram API. It only confirms that the service is running and accepts authenticated requests. Access is always protected by the same `X-Access-Token`.

```bash
curl -X GET http://localhost:8080/healthz \
  -H "X-Access-Token: your_api_access_token"
```

Response:

```json
{"status":"ok"}
```

If you override `PORT` or `HEALTHCHECK_ENDPOINT`, use those values in the URL.

---

## ✅ API Responses

### Proxy endpoint

- `204 No Content` — message was sent successfully.
- `400 Bad Request` — missing `chat_id` or `message`, or invalid JSON body.
- `401 Unauthorized` — missing or invalid `X-Access-Token`.
- `405 Method Not Allowed` — method is not `POST`.
- `500 Internal Server Error` — Telegram API request failed.

### Healthcheck

- `200 OK` — service is reachable and the token is valid.
- `401 Unauthorized` — missing or invalid `X-Access-Token`.
- `405 Method Not Allowed` — method is not `GET`.

---

## 📦 `.env` Example

```env
TELEGRAM_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
ACCESS_TOKEN=my-secret-token
PORT=8080
PROXY_ENDPOINT=/my/custom/endpoint
HEALTHCHECK_ENDPOINT=/internal/healthz
```

---

## 🛡️ Security Advice

- All working endpoints, including `healthcheck`, require a valid `X-Access-Token`.
- If you expose the service to a public subnet, consider adding IP whitelisting as an extra layer.
