package common

import (
	"github.com/gin-gonic/gin"
)

// R unified response structure
type R struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// OkData success response with data
func OkData(data any) R {
	return R{Code: 200, Message: "success", Data: data}
}

// Fail error response
func Fail(code int, msg string) R {
	return R{Code: code, Message: msg}
}

// WriteJSON write JSON response
func WriteJSON(c *gin.Context, status int, r R) {
	r.TraceID = c.GetHeader("X-Trace-ID")
	c.JSON(status, r)
}

// GetTraceID get trace ID from request
func GetTraceID(c *gin.Context) string {
	return c.GetHeader("X-Trace-ID")
}
