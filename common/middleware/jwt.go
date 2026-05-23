package middleware

import (
	"net/http"
	"shopmall/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
func AuthMiddleware() gin.HandlerFunc {
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

		c.Set("employee_id", int64(empID))
		c.Next()
	}
}
