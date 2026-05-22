package routes

import (
	"database/sql"
	"net/http"
	"shopmall/common"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// HealthChecker dependencies for health check
type HealthChecker struct {
	DB    *sql.DB
	Redis *redis.Client
}

// RegisterRoutes registers all routes
func RegisterRoutes(r *gin.Engine, hc *HealthChecker) {
	r.GET("/health", func(c *gin.Context) {
		if hc != nil && hc.DB != nil {
			if err := hc.DB.Ping(); err != nil {
				c.JSON(http.StatusServiceUnavailable, common.Fail(503, "database unavailable"))
				return
			}
		}
		if hc != nil && hc.Redis != nil {
			if err := hc.Redis.Ping(c.Request.Context()).Err(); err != nil {
				c.JSON(http.StatusServiceUnavailable, common.Fail(503, "redis unavailable"))
				return
			}
		}
		c.Status(http.StatusOK)
	})
}
