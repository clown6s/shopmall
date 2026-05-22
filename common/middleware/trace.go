package middleware

import (
	"fmt"
	"time"

	"shopmall/pkg"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TraceIDMiddleware generate/propagate X-Trace-ID
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" || !isValidTraceID(traceID) {
			traceID = fmt.Sprintf("local-%d", time.Now().UnixNano())
		}
		c.Header("X-Trace-ID", traceID)
		c.Set("trace_id", traceID)

		if pkg.Logger != nil {
			reqLogger := pkg.Logger.With(zap.String("trace_id", traceID))
			c.Set("logger", reqLogger)
		}

		c.Next()
	}
}

func isValidTraceID(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
