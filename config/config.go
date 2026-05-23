package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Host             string
	Port             string
	LivekitApiKey    string
	LivekitApiSecret string
	LivekitUrl       string
	S3KeyId          string
	S3KeySecret      string
	S3Endpoint       string
	S3Bucket         string
	S3Region         string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		Host:             getEnv("HOST", "0.0.0.0"),
		Port:             getEnv("PORT", "8080"),
		LivekitApiKey:    os.Getenv("LIVEKIT_API_KEY"),
		LivekitApiSecret: os.Getenv("LIVEKIT_API_SECRET"),
		LivekitUrl:       os.Getenv("LIVEKIT_URL"),
		S3KeyId:          os.Getenv("S3_KEY_ID"),
		S3KeySecret:      os.Getenv("S3_KEY_SECRET"),
		S3Endpoint:       os.Getenv("S3_ENDPOINT"),
		S3Bucket:         os.Getenv("S3_BUCKET"),
		S3Region:         os.Getenv("S3_REGION"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
