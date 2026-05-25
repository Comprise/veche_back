package auth

import (
	"encoding/json"

	"veche/internal/jsonrpc"
	"veche/internal/session"

	"github.com/gin-gonic/gin"
)

// Handler содержит JSON-RPC методы, общие для всех провайдеров авторизации.
// Логика конкретных провайдеров (Google, GitHub и т.д.) — в internal/auth/provider/.
type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Me — JSON-RPC метод "auth.me".
// Возвращает данные текущего пользователя из сессии (без обращения к БД).
func (h *Handler) Me(c *gin.Context, _ json.RawMessage) (any, *jsonrpc.Error) {
	data, _, ok := session.FromContext(c)
	if !ok {
		return nil, jsonrpc.ErrUnauthorized
	}
	return map[string]string{
		"user_id":      data.UserID,
		"email":        data.Email,
		"display_name": data.DisplayName,
		"provider":     data.Provider,
	}, nil
}

// Logout — JSON-RPC метод "auth.logout".
// Удаляет сессию из Redis — токен становится невалидным немедленно.
func (h *Handler) Logout(c *gin.Context, _ json.RawMessage) (any, *jsonrpc.Error) {
	_, token, ok := session.FromContext(c)
	if !ok {
		return nil, jsonrpc.ErrUnauthorized
	}
	if err := h.svc.Logout(c.Request.Context(), token); err != nil {
		return nil, jsonrpc.Internal("logout failed")
	}
	// MaxAge=-1 немедленно удаляет куки у веб-клиента.
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	return map[string]string{"status": "logged out"}, nil
}
