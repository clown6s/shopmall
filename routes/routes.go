package routes

import (
	"database/sql"
	"net/http"
	"shopmall/common"
	"shopmall/common/middleware"
	"shopmall/internal/handler"
	"shopmall/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// HealthChecker dependencies for health check
type HealthChecker struct {
	DB    *sql.DB
	Redis *redis.Client
}

// RegisterRoutes registers all routes
func RegisterRoutes(r *gin.Engine, hc *HealthChecker, empRepo *repository.EmployeeRepo, empHandler *handler.EmployeeHandler, deptHandler *handler.DepartmentHandler, prodHandler *handler.ProductHandler, roleHandler *handler.RoleHandler, permHandler *handler.PermissionHandler) {
	// 健康检查
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
		c.JSON(http.StatusOK, common.OkData(nil))
	})

	requireAdmin := middleware.RequireAdmin(empRepo)

	// API 路由组
	api := r.Group("/api")
	{
		// ── 不需要认证的路由 ──
		api.POST("/employee/login", empHandler.Login)

		auth := api.Group("/auth")
		{
			auth.POST("/refresh", empHandler.RefreshAccessToken)
			auth.POST("/logout", empHandler.Logout)
		}

		// ── 需要认证的路由 ──
		authed := api.Group("")
		authed.Use(middleware.AuthMiddleware())
		{
			// 员工管理
			emp := authed.Group("/employee")
			{
				emp.GET("/list", empHandler.List)
				emp.POST("", empHandler.Create)
				emp.PUT("/:id", empHandler.Update)
				emp.PUT("/:id/password", empHandler.ResetPassword)
			}

			// 当前用户
			user := authed.Group("/user")
			{
				user.GET("/me", empHandler.GetCurrentUser)
				user.GET("/menus", empHandler.GetUserMenus)
				user.PUT("/password", empHandler.ChangePassword)
			}

			// 部门管理
			dept := authed.Group("/department")
			{
				dept.GET("", deptHandler.GetAll)
				dept.GET("/all", deptHandler.GetAllSimple)
				dept.GET("/:deptId", deptHandler.GetByID)
				dept.POST("", deptHandler.Create)
				dept.PUT("/:deptId", deptHandler.Update)
				dept.DELETE("/:deptId", deptHandler.Delete)
			}

			// 商品（分类 + 品牌）
			product := authed.Group("/product")
			{
				product.GET("/category", prodHandler.CategoryTree)
				product.POST("/category", prodHandler.CategoryAdd)
				product.PUT("/category", prodHandler.CategoryEdit)
				product.DELETE("/category/:catId", prodHandler.CategoryDelete)
				product.GET("/brands", prodHandler.GetBrands)
				product.POST("/brand", prodHandler.AddBrand)
				product.PUT("/brand/:brandId", prodHandler.UpdateBrand)
				product.DELETE("/brand/:brandId", prodHandler.DeleteBrand)
			}

			// ── 管理员专用路由 ──
			admin := authed.Group("")
			admin.Use(requireAdmin)
			{
				// 角色管理
				role := admin.Group("/admin/role")
				{
					role.GET("", roleHandler.List)
					role.GET("/:id", roleHandler.GetByID)
					role.POST("", roleHandler.Create)
					role.PUT("/:id", roleHandler.Update)
					role.DELETE("/:id", roleHandler.Delete)
					role.GET("/:id/permissions", roleHandler.GetPermissionIDs)
					role.PUT("/:id/permissions", roleHandler.AssignPermissions)
				}

				// 权限管理
				perm := admin.Group("/admin/permission")
				{
					perm.GET("", permHandler.List)
					perm.GET("/:id", permHandler.GetByID)
					perm.POST("", permHandler.Create)
					perm.PUT("/:id", permHandler.Update)
					perm.DELETE("/:id", permHandler.Delete)
				}
			}
		}
	}
}
