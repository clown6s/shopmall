package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"shopmall/internal/config"

	"github.com/redis/go-redis/v9"
)

const maxRetries = 30
const retryInterval = 2 * time.Second

func InitRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.TLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(opts)

	var err error
	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = client.Ping(ctx).Err()
		cancel()
		if err == nil {
			return client, nil
		}
		log.Printf("[Redis] ping failed (retry %d/%d): %v", i, maxRetries, err)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("[Redis] failed after %d retries: %w", maxRetries, err)
}
