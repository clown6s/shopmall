package middleware

import (
	"fmt"
	"net/http"
	"shopmall/internal/service"
	"shopmall/pkg"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// GetEmployeeID 从 JWT 中间件设置的 context 中获取当前员工 ID。
func GetEmployeeID(c *gin.Context) int64 {
	if v, exists := c.Get("employee_id"); exists {
		switch id := v.(type) {
		case int64:
			return id
		case float64:
			return int64(id)
		case int:
			return int64(id)
		case string:
			if n, err := strconv.ParseInt(id, 10, 64); err == nil {
				return n
			}
		}
	}
	return 0
}

// AuthMiddleware JWT 认证中间件
// 从 httpOnly cookie "auth_token" 解析 JWT，将 employee_id 写入 gin.Context
// 同时校验 pw_version，确保密码修改后旧 token 失效
func AuthMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("auth_token")
		if err != nil || tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
			c.Abort()
			return
		}

		secret := service.GetJWTSecret()
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "token 无效或已过期"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "token 格式错误"})
			c.Abort()
			return
		}

		empID, ok := claims["employee_id"].(float64)
		if !ok || empID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "token 无效"})
			c.Abort()
			return
		}

		// 校验密码版本：密码修改后旧 token 应失效
		if rdb != nil {
			iatFloat, _ := claims["iat"].(float64)
			iat := int64(iatFloat)
			pwKey := fmt.Sprintf("pw_version:%d", int64(empID))
			pwVersion, err := rdb.Get(c.Request.Context(), pwKey).Int64()
			if err == nil && iat < pwVersion {
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "密码已修改，请重新登录"})
				c.Abort()
				return
			}
			if err != nil && err != redis.Nil {
				pkg.Logger.Error("[AuthMiddleware] pw_version Redis check failed, rejecting token",
					zap.Int64("empID", int64(empID)), zap.Error(err))
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "认证服务异常，请重新登录"})
				c.Abort()
				return
			}
		}

		c.Set("employee_id", int64(empID))
		c.Next()
	}
}
