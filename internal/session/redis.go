package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// sessionTTL — время жизни сессии в Redis.
// После истечения Redis автоматически удалит ключ.
const sessionTTL = 30 * 24 * time.Hour

// RedisStore — реализация Store на базе Redis.
type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Create генерирует случайный токен, сохраняет сессию в Redis и возвращает токен.
func (s *RedisStore) Create(ctx context.Context, data Data) (string, error) {
	// Генерируем 32 случайных байта из /dev/urandom (криптографически безопасно).
	// 32 байта = 256 бит = 2^256 возможных значений — перебрать нереально.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	// hex.EncodeToString превращает байты в строку из [0-9a-f].
	// 32 байта → 64 символа.
	token := hex.EncodeToString(b)

	// Сериализуем Data в JSON для хранения в Redis.
	payload, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}

	// SET session:<token> <payload> EX <ttl>
	if err := s.client.Set(ctx, "session:"+token, payload, sessionTTL).Err(); err != nil {
		return "", fmt.Errorf("redis set: %w", err)
	}
	return token, nil
}

// Get возвращает данные сессии по токену.
func (s *RedisStore) Get(ctx context.Context, token string) (*Data, error) {
	val, err := s.client.Get(ctx, "session:"+token).Bytes()
	if err == redis.Nil {
		// redis.Nil — специальная ошибка: ключ не существует или истёк.
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var data Data
	if err := json.Unmarshal(val, &data); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &data, nil
}

// Delete удаляет сессию из Redis.
// После этого Get вернёт ErrNotFound — токен немедленно инвалидируется.
func (s *RedisStore) Delete(ctx context.Context, token string) error {
	if err := s.client.Del(ctx, "session:"+token).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
