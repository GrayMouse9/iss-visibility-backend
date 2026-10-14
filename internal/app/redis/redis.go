package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"iss-visibility/internal/app/config"

	"github.com/go-redis/redis/v8"
)

const servicePrefix = "iss-visibility."

type Client struct {
	client *redis.Client
}

func New(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{client: client}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

// blacklistKey собирает ключ, под которым токен лежит в Redis.
// servicePrefix отделяет наши ключи от чужих, если база общая.
func blacklistKey(token string) string {
	return servicePrefix + "jwt.blacklist." + token
}

// WriteJWTToBlacklist помещает токен в чёрный список на время ttl.
// Значение пустое: важно само наличие ключа. По истечении ttl Redis удалит его сам —
// просроченный токен не пройдёт проверку срока и без чёрного списка.
func (c *Client) WriteJWTToBlacklist(ctx context.Context, token string, ttl time.Duration) error {
	if err := c.client.Set(ctx, blacklistKey(token), "", ttl).Err(); err != nil {
		return fmt.Errorf("failed to write token to blacklist: %w", err)
	}

	return nil
}

// CheckJWTInBlacklist сообщает, лежит ли токен в чёрном списке.
// Отсутствие ключа Redis отдаёт ошибкой redis.Nil — это не сбой, а ответ «нет».
func (c *Client) CheckJWTInBlacklist(ctx context.Context, token string) (bool, error) {
	err := c.client.Get(ctx, blacklistKey(token)).Err()

	switch {
	case errors.Is(err, redis.Nil):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("failed to check token in blacklist: %w", err)
	}

	return true, nil
}
