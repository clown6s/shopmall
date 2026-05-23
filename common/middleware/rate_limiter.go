package middleware

import (
	"context"
)

// NewLoginRateLimiter 创建登录速率限制器实例
func NewLoginRateLimiter() *LoginRateLimiter {
	return &LoginRateLimiter{}
}

// LoginRateLimiter 登录速率限制器。
// 当前为空实现桩，后续接入 Redis 滑动窗口计数器。
type LoginRateLimiter struct{}

// AllowCheck 检查该手机号是否允许登录尝试。
func (l *LoginRateLimiter) AllowCheck(_ context.Context, _ string) (allowed bool, remaining int, waitSeconds int) {
	return true, 5, 0
}

// RecordFail 记录一次登录失败。
func (l *LoginRateLimiter) RecordFail(_ context.Context, _ string) (int, int, error) {
	return 0, 5, nil
}

// GetRemainingAttempts 获取剩余尝试次数。
func (l *LoginRateLimiter) GetRemainingAttempts(_ context.Context, _ string) int {
	return 5
}

// ClearFail 清除失败记录。
func (l *LoginRateLimiter) ClearFail(_ context.Context, _ string) error {
	return nil
}
