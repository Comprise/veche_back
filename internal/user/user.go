// Пакет user отвечает за хранение и получение данных о пользователях.
// Единственное хранилище — SQLite через sqlc-сгенерированный код.
package user

import (
	"context"
	"time"
)

// User — доменная модель пользователя.
// Это не sqlc-модель (та хранит created_at как int64),
// а удобная Go-структура с нормальным time.Time.
type User struct {
	ID        string
	Email     string
	Name      string
	Provider  string
	CreatedAt time.Time
}

// Repository — интерфейс для работы с пользователями.
// Зависим от интерфейса, а не от конкретной реализации (SQLite).
// Это позволяет подменить хранилище в тестах или при миграции на PostgreSQL.
type Repository interface {
	// Upsert создаёт нового пользователя или обновляет email/имя существующего.
	// "Upsert" = "Update or Insert" — атомарная операция.
	// Вызывается при каждом логине: Google может изменить email пользователя.
	Upsert(ctx context.Context, id, email, name, provider string) (*User, error)

	// GetByID возвращает пользователя по его ID (Google Sub).
	GetByID(ctx context.Context, id string) (*User, error)
}
