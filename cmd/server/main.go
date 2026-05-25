package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"veche/internal/auth"
	googleprovider "veche/internal/auth/provider/google"
	"veche/internal/config"
	"veche/internal/livekit"
	"veche/internal/room"
	"veche/internal/server"
	"veche/internal/session"
	"veche/internal/user"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	// ── SQLite ──────────────────────────────────────────────────────────────
	db, err := sql.Open("sqlite3", cfg.DatabasePath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := runMigrations(cfg.DatabasePath); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// ── Redis ───────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping %s: %v", cfg.RedisAddr, err)
	}

	// ── Зависимости ─────────────────────────────────────────────────────────
	sessionStore := session.NewRedisStore(rdb)
	userRepo := user.NewSQLiteRepository(db)

	authSvc := auth.NewService(userRepo, sessionStore)

	// auth.Handler — общие методы (auth.me, auth.logout).
	authH := auth.NewHandler(authSvc)

	// Провайдеры авторизации — каждый получает auth.Service напрямую.
	// Чтобы добавить GitHub: github.NewHandler(cfg.GithubClientID, ..., authSvc, cfg.FrontendURL)
	googleH := googleprovider.NewHandler(
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirectURL,
		cfg.GoogleMobileRedirectURL,
		authSvc,
		cfg.FrontendURL,
	)

	// ── LiveKit ─────────────────────────────────────────────────────────────
	tokenSvc := livekit.NewTokenService(cfg.LivekitApiKey, cfg.LivekitApiSecret)
	egressSvc := livekit.NewEgressService(cfg.LivekitApiKey, cfg.LivekitApiSecret, cfg.LivekitUrl)
	roomH := room.NewHandler(tokenSvc, egressSvc)

	// ── HTTP сервер ─────────────────────────────────────────────────────────
	router := server.NewRouter(cfg, sessionStore, authH, googleH, roomH)
	srv := server.New(":"+cfg.Port, router)

	log.Printf("starting server on :%s", cfg.Port)
	if err := srv.Run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// runMigrations применяет все непримененные SQL-миграции из db/migrations/.
// Запускать из корня проекта: go run ./cmd/server/
func runMigrations(dbPath string) error {
	m, err := migrate.New("file://db/migrations", "sqlite3://"+dbPath)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
