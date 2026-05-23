package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	rateLimitPrefix    = "rate_limit:login:"
	rateLimitMax       = 5
	rateLimitWindowSec = 900 // 15 minutes
)

// NewLoginRateLimiter 创建登录速率限制器实例
func NewLoginRateLimiter(rdb *redis.Client) *LoginRateLimiter {
	return &LoginRateLimiter{rdb: rdb}
}

// LoginRateLimiter 登录速率限制器（Redis INCR + EXPIRE 滑动窗口）
type LoginRateLimiter struct {
	rdb *redis.Client
}

// AllowCheck 检查该手机号是否允许登录尝试（只读检查，不增加计数）。
func (l *LoginRateLimiter) AllowCheck(ctx context.Context, key string) (allowed bool, remaining int, waitSeconds int) {
	if l.rdb == nil {
		return true, rateLimitMax, 0
	}

	redisKey := rateLimitPrefix + key
	count, err := l.rdb.Get(ctx, redisKey).Int64()
	if err != nil {
		// Redis 故障或 key 不存在时允许
		return true, rateLimitMax, 0
	}

	if count >= int64(rateLimitMax) {
		ttl, _ := l.rdb.TTL(ctx, redisKey).Result()
		wait := int(ttl.Seconds())
		if wait < 0 {
			wait = 0
		}
		return false, 0, wait
	}

	return true, rateLimitMax - int(count), 0
}

// RecordFail 记录一次登录失败（唯一计数点）。
func (l *LoginRateLimiter) RecordFail(ctx context.Context, key string) (int, int, error) {
	if l.rdb == nil {
		return 0, rateLimitMax, nil
	}

	redisKey := rateLimitPrefix + key
	count, err := l.rdb.Incr(ctx, redisKey).Result()
	if err != nil {
		return 0, rateLimitMax, fmt.Errorf("redis incr failed: %w", err)
	}
	if count == 1 {
		l.rdb.Expire(ctx, redisKey, time.Duration(rateLimitWindowSec)*time.Second)
	}

	failCount := int(count) - 1 // 本次是第 count 次尝试，之前失败了 count-1 次
	remaining := rateLimitMax - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return failCount, remaining, nil
}

// GetRemainingAttempts 获取剩余尝试次数。
func (l *LoginRateLimiter) GetRemainingAttempts(ctx context.Context, key string) int {
	if l.rdb == nil {
		return rateLimitMax
	}

	redisKey := rateLimitPrefix + key
	count, err := l.rdb.Get(ctx, redisKey).Int64()
	if err != nil {
		return rateLimitMax
	}

	remaining := rateLimitMax - int(count)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ClearFail 清除失败记录。
func (l *LoginRateLimiter) ClearFail(ctx context.Context, key string) error {
	if l.rdb == nil {
		return nil
	}

	redisKey := rateLimitPrefix + key
	_, err := l.rdb.Del(ctx, redisKey).Result()
	return err
}
