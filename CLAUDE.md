# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Сборка
go build ./...

# Запуск (из корня проекта — golang-migrate ищет db/migrations/ относительно CWD)
go run ./cmd/server/

# Регенерация sqlc-кода после изменения db/queries/ или db/migrations/
~/go/bin/sqlc generate

# Сборка с CGO (требуется для go-sqlite3, gcc должен быть установлен)
CGO_ENABLED=1 go build ./...
```

## Архитектура

**Протокол:** JSON-RPC 2.0 через единственный эндпоинт `POST /rpc`. Исключения — OAuth-редиректы (`GET /auth/google/*`), которые требуют HTTP-редиректов и не могут быть JSON-RPC.

**Аутентификация:** opaque session token (32 байта `crypto/rand`, hex-encoded) хранится в Redis с TTL 30 дней. Никаких JWT. Веб получает токен через HttpOnly cookie `session_token`, мобильное — в теле ответа.

**Поток запроса:**
```
HTTP → middleware.ExtractSession (читает cookie/Bearer → Redis.Get) → session.SetContext
     → POST /rpc → jsonrpc.Server.Handle → метод → session.FromContext → ответ
```

**Слои и правило зависимостей:**

| Пакет | Зависит от |
|-------|-----------|
| `cmd/server` | все internal пакеты — единственное место DI |
| `internal/server` | `auth`, `auth/provider/*`, `room`, `session`, `jsonrpc`, `middleware` |
| `internal/auth` | `session`, `user` |
| `internal/auth/provider/google` | `auth` (Service интерфейс), `jsonrpc` |
| `internal/room` | `livekit`, `session`, `jsonrpc` |
| `internal/middleware` | `session` |
| `internal/user` | `dbsqlc` (сгенерировано sqlc) |
| `internal/session` | `redis/go-redis` |
| `internal/jsonrpc` | ничего внутреннего |
| `internal/livekit` | ничего внутреннего |
| `internal/dbsqlc` | сгенерировано — не редактировать вручную |

**Добавление нового JSON-RPC метода:**
1. Пишешь функцию с сигнатурой `func(c *gin.Context, params json.RawMessage) (any, *jsonrpc.Error)`
2. Регистрируешь одной строкой в `internal/server/router.go`: `rpc.Register("domain.method", handler.Method)`

**Добавление нового OAuth-провайдера (пример: GitHub):**
- Создаёшь `internal/auth/provider/github/handler.go`, пакет `github`
- Провайдер принимает `auth.Service` и `frontendURL` — не зависит от других провайдеров
- Регистрируешь маршруты в `router.go` и создаёшь хендлер в `main.go`

## База данных

**SQLite** (постоянное хранилище) + **Redis** (сессии).

- Миграции: `db/migrations/` в формате `NNNNNN_description.{up,down}.sql`. Применяются автоматически при старте через `golang-migrate`.
- SQL-запросы: `db/queries/` → `~/go/bin/sqlc generate` → `internal/dbsqlc/` (не редактировать).
- Конфиг sqlc: `sqlc.yaml`.
- SQLite требует CGO (`github.com/mattn/go-sqlite3`) → нужен `gcc`.

## Окружение

Копируй `.env.example` в `.env`. Обязательные переменные для запуска: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET`, `LIVEKIT_URL`. Redis должен быть запущен на `REDIS_ADDR` (по умолчанию `localhost:6379`).
