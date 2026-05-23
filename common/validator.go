package common

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ValidateLength 校验字符串长度（UTF-8 字符数）
func ValidateLength(value, field string, min, max int) error {
	n := utf8.RuneCountInString(value)
	if n < min {
		return fmt.Errorf("%s长度不能少于%d个字符", field, min)
	}
	if n > max {
		return fmt.Errorf("%s长度不能超过%d个字符", field, max)
	}
	return nil
}

// ValidateRequired 校验必填字段
func ValidateRequired(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s不能为空", field)
	}
	return nil
}

// ValidateEnum 校验枚举值
func ValidateEnum(value int8, field string, allowed ...int8) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s值无效", field)
}

// ValidateMethod 校验 HTTP 方法
func ValidateMethod(method string) error {
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "*":
		return nil
	}
	return fmt.Errorf("不支持的HTTP方法: %s", method)
}

// ValidateKeyFormat 校验键名格式（只允许字母、数字、下划线、冒号、连字符）
func ValidateKeyFormat(value, field string) error {
	for _, c := range value {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '_' || c == ':' || c == '-') {
			return fmt.Errorf("%s只能包含字母、数字、下划线、冒号和连字符", field)
		}
	}
	return nil
}
