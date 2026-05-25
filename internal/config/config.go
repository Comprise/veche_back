package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Host    string
	Port    string
	// FrontendURL используется для CORS и редиректа после OAuth.
	FrontendURL string

	// LiveKit — сервер для WebRTC звонков и записи.
	LivekitApiKey    string
	LivekitApiSecret string
	LivekitUrl       string

	// Google OAuth2 — авторизация через Google-аккаунт.
	GoogleClientID          string
	GoogleClientSecret      string
	GoogleRedirectURL       string      // для веб-клиента
	GoogleMobileRedirectURL string      // для мобильного приложения

	// RSA-ключи убраны: мы больше не используем JWT.
	// Сессии хранятся как opaque-токены в Redis.

	// DatabasePath — путь к файлу SQLite.
	// Например: "veche.db" или "/var/data/veche.db".
	DatabasePath string

	// RedisAddr — адрес Redis в формате "host:port".
	// Например: "localhost:6379".
	RedisAddr string

	// S3 — для хранения файлов записей (пока не реализовано).
	S3KeyId    string
	S3KeySecret string
	S3Endpoint  string
	S3Bucket    string
	S3Region    string
}

func Load() *Config {
	// godotenv.Load читает файл .env в текущей директории.
	// Если файла нет — ничего страшного, продолжаем с переменными окружения.
	godotenv.Load()

	return &Config{
		Host:                    getEnv("HOST", "0.0.0.0"),
		Port:                    getEnv("PORT", "8080"),
		FrontendURL:             os.Getenv("FRONTEND_URL"),
		LivekitApiKey:           os.Getenv("LIVEKIT_API_KEY"),
		LivekitApiSecret:        os.Getenv("LIVEKIT_API_SECRET"),
		LivekitUrl:              os.Getenv("LIVEKIT_URL"),
		GoogleClientID:          os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:      os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:       os.Getenv("GOOGLE_REDIRECT_URL"),
		GoogleMobileRedirectURL: os.Getenv("GOOGLE_MOBILE_REDIRECT_URL"),
		DatabasePath:            getEnv("DATABASE_PATH", "veche.db"),
		RedisAddr:               getEnv("REDIS_ADDR", "localhost:6379"),
		S3KeyId:                 os.Getenv("S3_KEY_ID"),
		S3KeySecret:             os.Getenv("S3_KEY_SECRET"),
		S3Endpoint:              os.Getenv("S3_ENDPOINT"),
		S3Bucket:                os.Getenv("S3_BUCKET"),
		S3Region:                os.Getenv("S3_REGION"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
