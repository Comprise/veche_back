// Пакет google реализует авторизацию через Google OAuth 2.0.
// Является одним из провайдеров авторизации.
//
// Чтобы добавить новый провайдер (например GitHub):
//
//	internal/auth/provider/github/handler.go — пакет github
//
// Каждый провайдер:
//   - принимает auth.Service (для создания сессии)
//   - принимает frontendURL (для редиректа после веб-логина)
//   - сам обрабатывает HTTP-ответ (куки / JSON)
//   - не зависит от других провайдеров
package google

import (
	"context"
	"encoding/json"
	"net/http"

	"veche/internal/auth"
	"veche/internal/jsonrpc"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google" // алиас чтобы не конфликтовать с именем пакета
	"google.golang.org/api/idtoken"
)

// Handler обрабатывает OAuth 2.0 поток Google и JSON-RPC метод обмена токенов.
type Handler struct {
	oauthConfig  *oauth2.Config // конфиг для веб-клиента
	mobileConfig *oauth2.Config // конфиг для мобильного приложения
	clientID     string
	authSvc      auth.Service // создание/удаление сессий
	frontendURL  string       // куда редиректить после веб-логина
}

// userInfo — ответ от Google userinfo API.
type userInfo struct {
	ID    string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewHandler(
	clientID, clientSecret, webRedirectURL, mobileRedirectURL string,
	authSvc auth.Service,
	frontendURL string,
) *Handler {
	makeCfg := func(redirect string) *oauth2.Config {
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirect,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: googleoauth.Endpoint,
		}
	}
	return &Handler{
		oauthConfig:  makeCfg(webRedirectURL),
		mobileConfig: makeCfg(mobileRedirectURL),
		clientID:     clientID,
		authSvc:      authSvc,
		frontendURL:  frontendURL,
	}
}

// LoginForWeb — GET /auth/google/login
// Перенаправляет браузер на страницу авторизации Google.
func (h *Handler) LoginForWeb(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, h.oauthConfig.AuthCodeURL("state-web"))
}

// LoginForMobile — GET /auth/google/mobile
// Возвращает URL авторизации — мобильное приложение открывает его само.
func (h *Handler) LoginForMobile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"auth_url": h.mobileConfig.AuthCodeURL("state-mobile")})
}

// Callback — GET /auth/google/callback
// Google перенаправляет сюда после авторизации пользователя.
func (h *Handler) Callback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	isMobile := c.Query("state") == "state-mobile"
	cfg := h.oauthConfig
	if isMobile {
		cfg = h.mobileConfig
	}

	oauthToken, err := cfg.Exchange(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token exchange failed"})
		return
	}

	resp, err := cfg.Client(c.Request.Context(), oauthToken).
		Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "userinfo failed"})
		return
	}
	defer resp.Body.Close()

	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decode failed"})
		return
	}

	if isMobile {
		h.finalizeForMobile(c, info.ID, info.Email, info.Name)
	} else {
		h.finalizeForWeb(c, info.ID, info.Email, info.Name)
	}
}

// TokenExchange — JSON-RPC метод "auth.google.tokenExchange"
// Для мобильного приложения: принимает Google ID Token и создаёт сессию.
func (h *Handler) TokenExchange(c *gin.Context, params json.RawMessage) (any, *jsonrpc.Error) {
	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(params, &req); err != nil || req.IDToken == "" {
		return nil, jsonrpc.InvalidParams("id_token required")
	}

	// Проверяем подпись ID Token у серверов Google.
	payload, err := idtoken.Validate(context.Background(), req.IDToken, h.clientID)
	if err != nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeUnauthorized, Message: "invalid id token"}
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	if name == "" {
		name = "User"
	}

	token, err := h.authSvc.Login(c.Request.Context(), payload.Subject, email, name, "google")
	if err != nil {
		return nil, jsonrpc.Internal("login failed")
	}

	return map[string]any{
		"session_token": token,
		"user": map[string]string{
			"id":           payload.Subject,
			"email":        email,
			"display_name": name,
			"provider":     "google",
		},
	}, nil
}

// finalizeForWeb создаёт сессию, ставит HttpOnly cookie и редиректит на фронтенд.
func (h *Handler) finalizeForWeb(c *gin.Context, googleID, email, name string) {
	token, err := h.authSvc.Login(c.Request.Context(), googleID, email, name, "google")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	c.SetCookie("session_token", token, 30*24*3600, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, h.frontendURL)
}

// finalizeForMobile создаёт сессию и возвращает токен в теле ответа.
func (h *Handler) finalizeForMobile(c *gin.Context, googleID, email, name string) {
	token, err := h.authSvc.Login(c.Request.Context(), googleID, email, name, "google")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session_token": token,
		"user": gin.H{
			"id":           googleID,
			"display_name": name,
			"provider":     "google",
		},
	})
}
