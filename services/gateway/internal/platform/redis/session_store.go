package redisstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/service"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string
	Password string
	DB       int
	Client   *redis.Client
}

type Store struct {
	client *redis.Client
}

func New(cfg Config) *Store {
	if cfg.Client == nil {
		cfg.Client = redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	}
	return &Store{client: cfg.Client}
}

func (s *Store) Save(ctx context.Context, entry service.GatewaySessionCacheEntry, ttl time.Duration) error {
	if ttl <= 0 {
		return service.ErrMalformedSession
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, service.SessionCacheKey(entry.AccessTokenHash), payload, ttl).Err()
}

func (s *Store) Get(ctx context.Context, accessTokenHash string) (service.GatewaySessionCacheEntry, error) {
	raw, err := s.client.Get(ctx, service.SessionCacheKey(accessTokenHash)).Bytes()
	if errors.Is(err, redis.Nil) {
		return service.GatewaySessionCacheEntry{}, service.ErrSessionNotFound
	}
	if err != nil {
		return service.GatewaySessionCacheEntry{}, err
	}
	var entry service.GatewaySessionCacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return service.GatewaySessionCacheEntry{}, service.ErrMalformedSession
	}
	return entry, nil
}

func (s *Store) Delete(ctx context.Context, accessTokenHash string) error {
	return s.client.Del(ctx, service.SessionCacheKey(accessTokenHash)).Err()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}
