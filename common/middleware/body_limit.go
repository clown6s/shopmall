package middleware

import (
	"io"
	"net/http"
	"shopmall/pkg"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MaxBodySize default max request body: 1MB
const MaxBodySize = 1 << 20

// BodyLimitMiddleware request body size limit (1MB default)
func BodyLimitMiddleware(maxBytes int) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = MaxBodySize
	}
	pkg.Logger.Info("[BodyLimit] max body size", zap.Int("bytes", maxBytes))
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(maxBytes))
		}
		c.Next()
	}
}

// ReadBody reads and limits request body
func ReadBody(c *gin.Context) ([]byte, error) {
	return io.ReadAll(c.Request.Body)
}
