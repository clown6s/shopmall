package middleware

import (
	"net/http"
	"shopmall/common"
	"shopmall/internal/config"
	"strings"

	"github.com/gin-gonic/gin"
)

type CORSOptions struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

// NewCORSOptions builds CORS config from config struct
func NewCORSOptions(c *config.CORSConfig) *CORSOptions {
	if c == nil {
		return &CORSOptions{
			AllowOrigins:     []string{"http://localhost:5173"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-XSRF-TOKEN"},
			AllowCredentials: true,
			MaxAge:           86400,
		}
	}
	return &CORSOptions{
		AllowOrigins:     c.AllowOrigins,
		AllowMethods:     c.AllowMethods,
		AllowHeaders:     c.AllowHeaders,
		AllowCredentials: c.AllowCredentials,
		MaxAge:           c.MaxAge,
	}
}

func CORSMiddleware(opts *CORSOptions) gin.HandlerFunc {
	if opts == nil {
		opts = &CORSOptions{
			AllowOrigins:     []string{"http://localhost:5173"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-XSRF-TOKEN"},
			AllowCredentials: true,
			MaxAge:           86400,
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// same-origin or non-browser, pass through
		if origin == "" {
			c.Next()
			return
		}

		allowed := false
		for _, o := range opts.AllowOrigins {
			if o == origin {
				allowed = true
				break
			}
		}

		if !allowed {
			common.WriteJSON(c, http.StatusForbidden, common.Fail(403, "Origin not allowed"))
			c.Abort()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")

		if opts.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// preflight
		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", strings.Join(opts.AllowMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(opts.AllowHeaders, ", "))
			c.Header("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
