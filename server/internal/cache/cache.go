// Package cache provides an optional Redis layer for hot find-references reads.
package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is a thin JSON cache over Redis. A nil Cache is a valid no-op, so the
// API and benchmarks can run with or without Redis configured.
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// New connects to Redis. If addr is empty it returns a nil cache (no-op).
func New(ctx context.Context, addr string, ttl time.Duration) (*Cache, error) {
	if addr == "" {
		return nil, nil
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Cache{client: client, ttl: ttl}, nil
}

// GetJSON loads a value into dst. ok is false on a cache miss.
func (c *Cache) GetJSON(ctx context.Context, key string, dst any) (ok bool, err error) {
	if c == nil {
		return false, nil
	}
	b, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return false, err
	}
	return true, nil
}

// SetJSON stores a value with the configured TTL.
func (c *Cache) SetJSON(ctx context.Context, key string, val any) error {
	if c == nil {
		return nil
	}
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, b, c.ttl).Err()
}

// Close releases the client.
func (c *Cache) Close() error {
	if c == nil {
		return nil
	}
	return c.client.Close()
}
