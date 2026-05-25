package server

import (
	"net/http"

	"veche/internal/auth"
	googleprovider "veche/internal/auth/provider/google"
	"veche/internal/config"
	"veche/internal/jsonrpc"
	"veche/internal/middleware"
	"veche/internal/room"
	"veche/internal/session"

	"github.com/gin-gonic/gin"
)

// NewRouter собирает Gin-роутер: CORS, middleware, маршруты.
//
// Архитектура маршрутов:
//   - GET /auth/google/*  — OAuth поток (нужны HTTP redirects, не JSON-RPC)
//   - POST /rpc           — весь остальной API через JSON-RPC 2.0
func NewRouter(
	cfg *config.Config,
	sessionStore session.Store,
	authH *auth.Handler,
	googleH *googleprovider.Handler,
	roomH *room.Handler,
) http.Handler {
	r := gin.Default()
	r.Use(corsMiddleware(cfg.FrontendURL))

	// Мягкое извлечение сессии: если токен есть — кладём данные в контекст.
	r.Use(middleware.ExtractSession(sessionStore))

	// OAuth маршруты — обычные HTTP GET (браузер делает redirects).
	r.GET("/auth/google/login", googleH.LoginForWeb)
	r.GET("/auth/google/callback", googleH.Callback)
	r.GET("/auth/google/mobile", googleH.LoginForMobile)

	// JSON-RPC диспетчер — единственная точка входа для всего API.
	rpc := jsonrpc.NewServer()

	rpc.Register("auth.google.tokenExchange", googleH.TokenExchange)
	rpc.Register("auth.me", authH.Me)
	rpc.Register("auth.logout", authH.Logout)
	rpc.Register("room.connectionDetails", roomH.GetConnectionDetails)
	rpc.Register("room.startRecording", roomH.StartRecording)
	rpc.Register("room.stopRecording", roomH.StopRecording)

	r.POST("/rpc", rpc.Handle)
	return r
}

func corsMiddleware(frontendURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")
		c.Header("Access-Control-Allow-Origin", frontendURL)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
