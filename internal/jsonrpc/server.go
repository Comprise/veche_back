package jsonrpc

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Request — входящий JSON-RPC 2.0 запрос.
//
// Пример:
//
//	{"jsonrpc":"2.0","method":"auth.me","params":{},"id":1}
//
// json.RawMessage для Params означает "оставить как есть" — каждый метод
// сам распарсит параметры в свою структуру через json.Unmarshal.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	// ID может быть числом, строкой или null. any (= interface{}) подходит для всего.
	ID any `json:"id"`
}

// Response — ответ JSON-RPC 2.0.
//
// Пример успеха:   {"jsonrpc":"2.0","result":{...},"id":1}
// Пример ошибки:   {"jsonrpc":"2.0","error":{"code":-32601,"message":"..."},"id":1}
//
// omitempty: в ответе будет либо "result", либо "error" — никогда оба.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
	ID      any    `json:"id"`
}

// MethodFunc — сигнатура каждого зарегистрированного метода.
//
// c *gin.Context — HTTP-контекст: куки, заголовки, IP клиента и т.д.
// params — сырые параметры из JSON-запроса, метод парсит их сам.
// Возвращает (результат, nil) при успехе или (nil, *Error) при ошибке.
type MethodFunc func(c *gin.Context, params json.RawMessage) (any, *Error)

// Server — диспетчер JSON-RPC. Хранит карту имён методов → функций.
type Server struct {
	methods map[string]MethodFunc
}

func NewServer() *Server {
	return &Server{methods: make(map[string]MethodFunc)}
}

// Register добавляет метод в реестр диспетчера.
//
// Пример:
//
//	rpc.Register("auth.me", authHandler.Me)
//	rpc.Register("room.connectionDetails", roomHandler.GetConnectionDetails)
func (s *Server) Register(name string, fn MethodFunc) {
	s.methods[name] = fn
}

// Handle — Gin-хендлер для POST /rpc.
// Весь API проходит через одну точку входа — это и есть JSON-RPC.
func (s *Server) Handle(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		// Не удалось распарсить JSON — тело запроса невалидное.
		respond(c, nil, ErrParseError)
		return
	}

	// JSON-RPC 2.0 требует поле "jsonrpc":"2.0".
	if req.JSONRPC != "2.0" {
		respond(c, req.ID, ErrInvalidRequest)
		return
	}

	fn, ok := s.methods[req.Method]
	if !ok {
		respond(c, req.ID, ErrMethodNotFound)
		return
	}

	result, rpcErr := fn(c, req.Params)
	if rpcErr != nil {
		respond(c, req.ID, rpcErr)
		return
	}

	// Notification — запрос без "id". По спецификации ответ не нужен.
	if req.ID == nil {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	})
}

// respond отправляет ответ с ошибкой.
// По спецификации JSON-RPC HTTP-статус всегда 200, даже для ошибок —
// статус ошибки передаётся внутри JSON в поле "error".
func respond(c *gin.Context, id any, err *Error) {
	c.JSON(http.StatusOK, Response{
		JSONRPC: "2.0",
		Error:   err,
		ID:      id,
	})
}
