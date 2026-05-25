// Пакет session отвечает за хранение сессий пользователей.
// Сессия — это пара "токен ↔ данные пользователя" хранящаяся в Redis.
// Токен — случайная 64-символьная hex-строка (32 байта crypto/rand).
package session

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
)

// ErrNotFound возвращается из Store.Get когда сессия не найдена:
// токен истёк, невалидный, или был удалён при логауте.
var ErrNotFound = errors.New("session not found")

// Data — всё что мы храним в Redis для каждой сессии.
// Эти данные дублируют часть записи из таблицы users, чтобы каждый
// запрос не ходил в SQLite. Изменения в users.name/email отражаются
// только при следующем логине (пересоздании сессии).
type Data struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Provider    string `json:"provider"`
}

// Store — интерфейс хранилища сессий.
// Сейчас реализован через Redis (redis.go), но при необходимости
// можно написать другую реализацию — например, in-memory для тестов.
type Store interface {
	// Create генерирует криптографически случайный токен,
	// сохраняет Data в хранилище и возвращает токен.
	Create(ctx context.Context, data Data) (token string, err error)
	// Get возвращает Data по токену. Возвращает ErrNotFound если токен
	// не существует или истёк.
	Get(ctx context.Context, token string) (*Data, error)
	// Delete удаляет сессию. После этого токен немедленно становится невалидным.
	// Используется при логауте.
	Delete(ctx context.Context, token string) error
}

// Приватные ключи для gin.Context — чтобы другие пакеты использовали
// только наши хелперы SetContext / FromContext.
const (
	ctxKeyData  = "session_data"
	ctxKeyToken = "session_token_val"
)

// SetContext сохраняет данные сессии в контексте текущего HTTP-запроса.
// Вызывается middleware после успешной проверки токена.
func SetContext(c *gin.Context, data *Data, token string) {
	c.Set(ctxKeyData, data)
	c.Set(ctxKeyToken, token)
}

// FromContext извлекает данные сессии из контекста запроса.
// Возвращает (nil, "", false) если сессии нет (неавторизованный запрос).
// Хендлеры вызывают эту функцию в начале — если ok=false, возвращают ErrUnauthorized.
func FromContext(c *gin.Context) (data *Data, token string, ok bool) {
	raw, exists := c.Get(ctxKeyData)
	if !exists {
		return nil, "", false
	}
	data, ok = raw.(*Data)
	if !ok {
		return nil, "", false
	}
	token = c.GetString(ctxKeyToken)
	return data, token, true
}
