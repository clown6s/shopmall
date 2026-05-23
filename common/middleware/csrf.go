package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

// SetCSRFCookie 生成 CSRF token 并设置为非 httpOnly Cookie（供前端 JS 读取）。
// token 通过 cookie 下发，前端在 POST/PUT/DELETE 请求时通过 X-XSRF-TOKEN header 带回。
// ctx 参数保留兼容签名，当前未使用。
func SetCSRFCookie(_ interface{}, c *gin.Context, _ string) {
	token := generateCSRFToken()
	secure := false
	if strings.EqualFold(c.Request.Header.Get("X-Forwarded-Proto"), "https") {
		secure = true
	}
	// SameSite=Lax：允许 GET 导航携带，阻止跨站 POST 携带
	c.SetSameSite(3)                                                // http.SameSiteLax
	c.SetCookie("xsrf-token", token, 86400, "/", "", secure, false) // httpOnly=false
}

// ValidateCSRF 验证请求中的 X-XSRF-TOKEN header 是否与 cookie 中的 xsrf-token 匹配。
// 用于中间件，保护所有写操作（POST/PUT/DELETE/PATCH）。
func ValidateCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			c.Next()
			return
		}

		cookieToken, err := c.Cookie("xsrf-token")
		if err != nil || cookieToken == "" {
			c.JSON(403, gin.H{"code": 403, "message": "CSRF token 缺失"})
			c.Abort()
			return
		}

		headerToken := c.GetHeader("X-XSRF-TOKEN")
		if headerToken == "" || !strings.EqualFold(headerToken, cookieToken) {
			c.JSON(403, gin.H{"code": 403, "message": "CSRF token 不匹配"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// InvalidateCSRF 清除 CSRF cookie（登出时调用）。
func InvalidateCSRF(_ string) {
	// 当前 CSRF 是 stateless cookie，清 cookie 在 handler 层通过 c.SetCookie("xsrf-token", "", -1, ...) 完成
	// 此函数保留签名兼容
}

// generateCSRFToken 生成 32 字节随机 hex token（64 字符）
func generateCSRFToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
