# 📬 Telegram Proxy Bot Service

Сервис-прокси для отправки сообщений в Telegram через вашего бота.  
Принимает HTTP-запросы и пересылает сообщения в указанный `chat_id`.

---

## ⚙️ Переменные окружения

В файле `.env` необходимо указать:

```env
TELEGRAM_TOKEN=your_telegram_bot_token
ACCESS_TOKEN=your_api_access_token
```

- `TELEGRAM_TOKEN` — токен Telegram-бота.
- `ACCESS_TOKEN` — токен для доступа к API (передаётся в заголовке запроса).

---

## 🚀 Запуск

### 1. Сборка Docker-образа

```bash
docker build -t telegram-proxy .
```

### 2. Запуск контейнера

```bash
docker run -p 8080:8080 --env-file .env -d telegram-proxy
```

---

## 🔗 Использование

```bash
curl -X POST http://localhost:8080/service/proxy/telegram \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: your_api_access_token" \
  -d '{"chat_id": "123456789", "message": "Сообщение"}'
```

### Параметры запроса

| Поле     | Тип     | Описание                     |
|----------|---------|------------------------------|
| chat_id  | string  | ID чата или пользователя     |
| message  | string  | Текст сообщения              |

---

## ✅ Ответы

- `204 No content` — сообщение успешно отправлено
- `401 Unauthorized` — неверный `X-Access-Token`
- `400 Bad Request` — отсутствует `chat_id` или `message`
- `500 Internal Server Error` — ошибка при отправке сообщения в Telegram

---

## 📦 Пример `.env`

```env
TELEGRAM_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
ACCESS_TOKEN=my-secret-token
```

---

## 🛡️ Безопасность

- Убедитесь, что `.env` **не загружен в репозиторий** (`.gitignore`).
- При необходимости можно ограничить доступ к API по IP.