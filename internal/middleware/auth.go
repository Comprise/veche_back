// Пакет middleware содержит Gin-middleware для обработки HTTP-запросов.
package middleware

import (
	"strings"

	"veche/internal/session"

	"github.com/gin-gonic/gin"
)

// ExtractSession — Gin middleware для извлечения и валидации сессии.
//
// Работает "мягко" (soft): если токен не найден или невалидный — запрос
// продолжается без сессии. Каждый JSON-RPC метод сам решает, нужна ли ему
// аутентификация, вызывая session.FromContext(c).
//
// Это лучше чем "жёсткий" middleware (который возвращает 401) потому что
// у нас один эндпоинт /rpc, где одни методы публичные, другие — защищённые.
func ExtractSession(store session.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.Next() // нет токена — продолжаем без сессии
			return
		}

		// Ищем сессию в Redis. Если истекла или невалидная — Get вернёт ErrNotFound.
		data, err := store.Get(c.Request.Context(), token)
		if err != nil {
			c.Next() // токен невалидный — продолжаем без сессии
			return
		}

		// Сохраняем данные сессии в контексте запроса.
		// Хендлеры читают их через session.FromContext(c).
		session.SetContext(c, data, token)
		c.Next()
	}
}

// extractToken ищет токен сессии в запросе.
// Порядок поиска: куки (веб) → Authorization header (мобильное приложение).
func extractToken(c *gin.Context) string {
	// Вариант 1: HttpOnly куки (Svelte фронтенд)
	if cookie, err := c.Cookie("session_token"); err == nil && cookie != "" {
		return cookie
	}
	// Вариант 2: Bearer токен в заголовке (мобильное приложение)
	// Формат: "Authorization: Bearer <token>"
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
