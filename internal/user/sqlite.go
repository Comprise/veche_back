package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"veche/internal/dbsqlc"
)

// SQLiteRepository — реализация Repository поверх sqlc-сгенерированного кода.
// Принимает *sql.DB и создаёт sqlc.Queries для выполнения SQL-запросов.
type SQLiteRepository struct {
	q *dbsqlc.Queries
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	// dbsqlc.New оборачивает *sql.DB в структуру Queries с методами для каждого запроса.
	return &SQLiteRepository{q: dbsqlc.New(db)}
}

// Upsert сохраняет пользователя в БД (создаёт или обновляет).
func (r *SQLiteRepository) Upsert(ctx context.Context, id, email, name, provider string) (*User, error) {
	u, err := r.q.UpsertUser(ctx, dbsqlc.UpsertUserParams{
		ID:       id,
		Email:    email,
		Name:     name,
		Provider: provider,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return toUser(u), nil
}

// GetByID возвращает пользователя по ID.
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user %q not found", id)
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return toUser(u), nil
}

// toUser конвертирует sqlc-модель в доменную структуру User.
// В SQLite created_at хранится как Unix-timestamp (int64 — секунды с 01.01.1970).
// time.Unix(sec, 0) превращает его в time.Time в UTC.
func toUser(u dbsqlc.User) *User {
	return &User{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Provider:  u.Provider,
		CreatedAt: time.Unix(u.CreatedAt, 0).UTC(),
	}
}
