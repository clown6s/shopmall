package middleware

import (
	"net/http"
	"runtime/debug"
	"shopmall/common"
	"shopmall/pkg"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				pkg.Logger.Error("[PanicRecovery] panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("stack", string(stack)),
				)
				common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
				c.Abort()
			}
		}()
		c.Next()
	}
}
