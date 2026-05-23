package service

import (
	"errors"
	"strings"
)

// isDuplicateKeyError 判断错误是否为数据库唯一约束冲突（MySQL 1062）
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Duplicate entry") ||
		strings.Contains(err.Error(), "1062") ||
		errors.Is(err, ErrDuplicateEntry)
}

// ErrDuplicateEntry 数据库唯一约束冲突错误（由 handler 层匹配使用）
var ErrDuplicateEntry = errors.New("数据已存在")
