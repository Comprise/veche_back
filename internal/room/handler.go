// Пакет room содержит хендлеры для работы с комнатами LiveKit:
// получение токена для подключения и управление записью.
package room

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"veche/internal/jsonrpc"
	"veche/internal/livekit"
	"veche/internal/session"

	"github.com/gin-gonic/gin"
)

// Handler содержит JSON-RPC методы для работы с комнатами.
type Handler struct {
	tokenizer livekit.Tokenizer // выдаёт JWT-токены для подключения к LiveKit
	recorder  livekit.Recorder  // управляет записью (egress)
}

func NewHandler(tokenizer livekit.Tokenizer, recorder livekit.Recorder) *Handler {
	return &Handler{tokenizer: tokenizer, recorder: recorder}
}

// GetConnectionDetails — JSON-RPC метод "room.connectionDetails"
// Возвращает LiveKit-токен для подключения к комнате.
//
// Пример вызова:
//
//	{"jsonrpc":"2.0","method":"room.connectionDetails","params":{"room_name":"my-room","participant_name":"Alice"},"id":1}
func (h *Handler) GetConnectionDetails(c *gin.Context, params json.RawMessage) (any, *jsonrpc.Error) {
	_, _, ok := session.FromContext(c)
	if !ok {
		return nil, jsonrpc.ErrUnauthorized
	}

	var req struct {
		RoomName        string `json:"room_name"`
		ParticipantName string `json:"participant_name"`
		Metadata        string `json:"metadata,omitempty"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, jsonrpc.InvalidParams("invalid params")
	}
	if req.RoomName == "" || req.ParticipantName == "" {
		return nil, jsonrpc.InvalidParams("room_name and participant_name are required")
	}

	// Постфикс обеспечивает уникальность identity в LiveKit при открытии
	// нескольких вкладок с одним именем участника.
	postfix, err := c.Cookie("participant_postfix")
	if err != nil || postfix == "" {
		postfix, err = randomString(4)
		if err != nil {
			return nil, jsonrpc.Internal("failed to generate postfix")
		}
		c.SetCookie("participant_postfix", postfix, 7200, "/", "", false, true)
	}

	token, err := h.tokenizer.BuildConnectionDetails(
		req.RoomName,
		req.ParticipantName,
		req.Metadata,
		req.ParticipantName+"__"+postfix, // уникальный identity для LiveKit
	)
	if err != nil {
		return nil, jsonrpc.Internal("failed to generate token")
	}

	return map[string]string{
		"room_name":         req.RoomName,
		"participant_name":  req.ParticipantName,
		"participant_token": token,
	}, nil
}

// StartRecording — JSON-RPC метод "room.startRecording"
// Запускает запись аудио комнаты через LiveKit egress.
func (h *Handler) StartRecording(c *gin.Context, params json.RawMessage) (any, *jsonrpc.Error) {
	_, _, ok := session.FromContext(c)
	if !ok {
		return nil, jsonrpc.ErrUnauthorized
	}

	var req struct {
		RoomName string `json:"room_name"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.RoomName == "" {
		return nil, jsonrpc.InvalidParams("room_name required")
	}

	if err := h.recorder.StartRecording(req.RoomName); err != nil {
		if errors.Is(err, livekit.ErrRecordingAlreadyActive) {
			return nil, &jsonrpc.Error{Code: jsonrpc.CodeConflict, Message: "recording already active"}
		}
		return nil, jsonrpc.Internal(fmt.Sprintf("start recording: %v", err))
	}
	return map[string]string{"status": "recording started"}, nil
}

// StopRecording — JSON-RPC метод "room.stopRecording"
// Останавливает активную запись комнаты.
func (h *Handler) StopRecording(c *gin.Context, params json.RawMessage) (any, *jsonrpc.Error) {
	_, _, ok := session.FromContext(c)
	if !ok {
		return nil, jsonrpc.ErrUnauthorized
	}

	var req struct {
		RoomName string `json:"room_name"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.RoomName == "" {
		return nil, jsonrpc.InvalidParams("room_name required")
	}

	if err := h.recorder.StopRecording(req.RoomName); err != nil {
		if errors.Is(err, livekit.ErrNoActiveRecording) {
			return nil, &jsonrpc.Error{Code: jsonrpc.CodeNotFound, Message: "no active recording"}
		}
		return nil, jsonrpc.Internal(fmt.Sprintf("stop recording: %v", err))
	}
	return map[string]string{"status": "recording stopped"}, nil
}

// randomString генерирует случайную строку длиной n символов из [a-z0-9].
// Использует crypto/rand — криптографически безопасный генератор (читает /dev/urandom).
// В отличие от math/rand, нельзя предсказать следующее значение.
func randomString(n int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	for i := range b {
		// % len(letters) даёт индекс в диапазоне [0, 35].
		// Небольшое смещение (modulo bias) несущественно для 4-символьного постфикса.
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b), nil
}
