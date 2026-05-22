package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestTimeout default handler context timeout
const RequestTimeout = 8 * time.Second

// ReqCtx returns a cancellable context for the request
func ReqCtx(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), RequestTimeout)
}
