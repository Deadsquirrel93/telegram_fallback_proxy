# 📬 Telegram Fallback Proxy Bot Service

[🇷🇺 Русский](Readme.md) | [🇬🇧 English](Readme_en.md)

Сервис-прокси для отправки сообщений в Telegram через вашего бота.  
Принимает HTTP-запросы и пересылает сообщения в указанный `chat_id`.

**Для чего это нужно?**  
1. **Стабильность отправки (Fallback):** Проект изначально создавался как "запасной маршрут" для тех случаев, когда основные серверы проекта сталкивались с нестабильностью сети или таймаутами при обращении к `api.telegram.org` (например, из-за сетевых ограничений или особенностей провайдера). Развернув этот легковесный прокси на узле со стабильным доступом к интернету, вы обеспечиваете надежную и гарантированную доставку всех Telegram-уведомлений.
2. **Безопасность токенов:** Это отличный паттерн для скрытия настоящего токена Telegram-бота от клиентских приложений (frontend, mobile apps) или других микросервисов. Вы централизуете логику уведомлений в одном месте и общаетесь с прокси по собственному `ACCESS_TOKEN`.

## 🌟 Особенности (Features)

- 🪶 **Супер-легковесный Docker-образ** (~27 МБ) благодаря multi-stage сборке.
- ⏱️ **Встроенные таймауты HTTP-сервера**, обеспечивающие защиту от DDoS (Slowloris attacks).
- 🛡️ **Защита по памяти** с ограничением размера входящего JSON-запроса (до 1 MB).
- 🔄 **Переиспользование HTTP-клиента** с таймаутами для внешних вызовов к Telegram API, предотвращающее утечки горутин.
- ❤️ **Защищённый healthcheck** с обязательной авторизацией по `X-Access-Token`.
- ⚙️ **Гибкая конфигурация**: порт, эндпоинты и доступы легко настраиваются через переменные окружения.

---

## ⚙️ Переменные окружения

Сервис может читать настройки из `.env` или из системных переменных окружения. Минимально обязательны только `TELEGRAM_TOKEN` и `ACCESS_TOKEN`.

```env
TELEGRAM_TOKEN=your_telegram_bot_token
ACCESS_TOKEN=your_api_access_token
PORT=8080
PROXY_ENDPOINT=/service/proxy/telegram
HEALTHCHECK_ENDPOINT=/healthz
```

- `TELEGRAM_TOKEN` — токен Telegram-бота.
- `ACCESS_TOKEN` — токен для доступа к API (передаётся в заголовке запроса).
- `PORT` — порт HTTP-сервера внутри контейнера/процесса, по умолчанию `8080`.
- `PROXY_ENDPOINT` — эндпоинт (URI путь), по умолчанию используется `/service/proxy/telegram`.
- `HEALTHCHECK_ENDPOINT` — эндпоинт проверки состояния, по умолчанию `/healthz`.

---

## 🚀 Запуск

### 1. Сборка Docker-образа

```bash
docker build -t telegram-proxy .
```

### 2. Запуск контейнера

```bash
docker run -d \
  --name telegram-proxy \
  --restart unless-stopped \
  -p 8080:8080 \
  --env-file .env \
  telegram-proxy
```

`--restart unless-stopped` автоматически поднимет контейнер после падения или перезапуска Docker daemon.

Если меняете `PORT`, пробрасывайте тот же порт и снаружи:

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

## 🔗 Использование

### Отправка сообщения

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

## ❤️ Healthcheck

Healthcheck не отправляет запросы в Telegram API и только подтверждает, что сервис запущен и принимает авторизованные запросы. Доступ к нему всегда защищён тем же `X-Access-Token`.

```bash
curl -X GET http://localhost:8080/healthz \
  -H "X-Access-Token: your_api_access_token"
```

Ответ:

```json
{"status":"ok"}
```

Если вы переопределили `PORT` или `HEALTHCHECK_ENDPOINT`, используйте их значения в URL.

---

## ✅ Ответы API

### Прокси-эндпоинт

- `204 No Content` — сообщение успешно отправлено.
- `400 Bad Request` — отсутствует `chat_id` или `message`, либо передан невалидный JSON.
- `401 Unauthorized` — неверный или отсутствующий `X-Access-Token`.
- `405 Method Not Allowed` — используется не `POST`.
- `500 Internal Server Error` — ошибка при отправке сообщения в Telegram.

### Healthcheck

- `200 OK` — сервис доступен и токен валиден.
- `401 Unauthorized` — неверный или отсутствующий `X-Access-Token`.
- `405 Method Not Allowed` — используется не `GET`.

---

## 📦 Пример `.env`

```env
TELEGRAM_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
ACCESS_TOKEN=my-secret-token
PORT=8080
PROXY_ENDPOINT=/my/custom/endpoint
HEALTHCHECK_ENDPOINT=/internal/healthz
```

---

## 🛡️ Безопасность

- Все рабочие эндпоинты, включая `healthcheck`, требуют корректный заголовок `X-Access-Token`.
- При необходимости можно дополнительно ограничить доступ к API по IP.
