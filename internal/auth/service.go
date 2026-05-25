// Пакет auth содержит бизнес-логику аутентификации и авторизации.
// Вместо JWT используем opaque session token: случайная строка,
// которую клиент хранит в куки (веб) или Keychain/Keystore (мобильное),
// а сервер хранит в Redis с привязкой к данным пользователя.
//
// Плюсы opaque token перед JWT:
//   - Мгновенный logout: просто удаляем ключ из Redis
//   - Не надо управлять refresh-токенами
//   - Легко видеть все активные сессии (admin-панель)
//   - Нет риска "утечки" данных через payload токена
package auth

import (
	"context"
	"fmt"

	"veche/internal/session"
	"veche/internal/user"
)

// Service — бизнес-логика аутентификации.
// Объединяет работу с пользователями (SQLite) и сессиями (Redis).
type Service interface {
	// Login сохраняет пользователя в БД (upsert) и создаёт новую сессию в Redis.
	// Возвращает opaque session token для передачи клиенту.
	Login(ctx context.Context, googleID, email, name, provider string) (string, error)
	// Logout немедленно удаляет сессию из Redis.
	// После этого Get(token) вернёт session.ErrNotFound.
	Logout(ctx context.Context, token string) error
}

type authService struct {
	users    user.Repository  // SQLite: постоянное хранилище пользователей
	sessions session.Store    // Redis: быстрое хранилище активных сессий
}

func NewService(users user.Repository, sessions session.Store) Service {
	return &authService{users: users, sessions: sessions}
}

func (s *authService) Login(ctx context.Context, googleID, email, name, provider string) (string, error) {
	// Сохраняем или обновляем пользователя в SQLite.
	// При каждом логине Google может изменить имя или email.
	u, err := s.users.Upsert(ctx, googleID, email, name, provider)
	if err != nil {
		return "", fmt.Errorf("upsert user: %w", err)
	}

	// Создаём сессию в Redis. Данные пользователя кешируются прямо в сессии,
	// чтобы не ходить в SQLite при каждом запросе.
	token, err := s.sessions.Create(ctx, session.Data{
		UserID:      u.ID,
		Email:       u.Email,
		DisplayName: u.Name,
		Provider:    u.Provider,
	})
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

func (s *authService) Logout(ctx context.Context, token string) error {
	if err := s.sessions.Delete(ctx, token); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
