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
func RegisterRoutes(r *gin.Engine, hc *HealthChecker, empRepo *repository.EmployeeRepo, roleRepo *repository.RoleRepo, permRepo *repository.PermissionRepo, empHandler *handler.EmployeeHandler, deptHandler *handler.DepartmentHandler, prodHandler *handler.ProductHandler, roleHandler *handler.RoleHandler, permHandler *handler.PermissionHandler) {
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

	// ── 权限中间件工厂 ──
	// perm 创建一个闭包，传入所需的权限 key 即可
	perm := func(keys ...string) gin.HandlerFunc {
		return middleware.RequirePermission(empRepo, roleRepo, permRepo, keys...)
	}

	// API 路由组
	api := r.Group("/api")
	{
		// ── 不需要认证的路由 ──
		api.POST("/employee/login", empHandler.Login)

		auth := api.Group("/auth")
		{
			auth.POST("/refresh", middleware.ValidateCSRF(), empHandler.RefreshAccessToken)
			auth.POST("/logout", middleware.ValidateCSRF(), empHandler.Logout)
		}

		// ── 需要认证的路由 ──
		authed := api.Group("")
		authed.Use(middleware.AuthMiddleware(hc.Redis))
		authed.Use(middleware.ValidateCSRF())
		{
			// ── 当前用户（白名单，不需要额外权限） ──
			user := authed.Group("/user")
			{
				user.GET("/me", empHandler.GetCurrentUser)
				user.GET("/menus", empHandler.GetUserMenus)
				user.PUT("/password", empHandler.ChangePassword)
			}

			// ── 员工管理 ──
			emp := authed.Group("/employee")
			{
				emp.GET("/list", perm("system:employee:list"), empHandler.List)
				emp.POST("", perm("system:employee:add"), empHandler.Create)
				emp.PUT("/:id", perm("system:employee:update"), empHandler.Update)
				emp.PUT("/:id/password", perm("system:employee:reset_pwd"), empHandler.ResetPassword)
			}

			// ── 部门管理 ──
			dept := authed.Group("/department")
			{
				dept.GET("", perm("system:department:list"), deptHandler.GetAll)
				dept.GET("/all", perm("system:department:list"), deptHandler.GetAllSimple)
				dept.GET("/:deptId", perm("system:department:list"), deptHandler.GetByID)
				dept.POST("", perm("system:department:add"), deptHandler.Create)
				dept.PUT("/:deptId", perm("system:department:update"), deptHandler.Update)
				dept.DELETE("/:deptId", perm("system:department:delete"), deptHandler.Delete)
			}

			// ── 商品（分类 + 品牌） ──
			product := authed.Group("/product")
			{
				product.GET("/category", perm("product:category:list"), prodHandler.CategoryTree)
				product.POST("/category", perm("product:category:add"), prodHandler.CategoryAdd)
				product.PUT("/category", perm("product:category:update"), prodHandler.CategoryEdit)
				product.DELETE("/category/:catId", perm("product:category:delete"), prodHandler.CategoryDelete)
				product.GET("/brands", perm("product:brand:list"), prodHandler.GetBrands)
				product.POST("/brand", perm("product:brand:add"), prodHandler.AddBrand)
				product.PUT("/brand/:brandId", perm("product:brand:update"), prodHandler.UpdateBrand)
				product.DELETE("/brand/:brandId", perm("product:brand:delete"), prodHandler.DeleteBrand)
			}

			// ── 角色管理 ──
			role := authed.Group("/admin/role")
			{
				role.GET("", perm("system:role:list"), roleHandler.List)
				role.GET("/:id", perm("system:role:list"), roleHandler.GetByID)
				role.POST("", perm("system:role:add"), roleHandler.Create)
				role.PUT("/:id", perm("system:role:update"), roleHandler.Update)
				role.DELETE("/:id", perm("system:role:delete"), roleHandler.Delete)
				role.GET("/:id/permissions", perm("system:role:list"), roleHandler.GetPermissionIDs)
				role.PUT("/:id/permissions", perm("system:role:assign_perm"), roleHandler.AssignPermissions)
			}

			// ── 权限管理 ──
			prm := authed.Group("/admin/permission")
			{
				prm.GET("", perm("system:permission:list"), permHandler.List)
				prm.GET("/:id", perm("system:permission:list"), permHandler.GetByID)
				prm.POST("", perm("system:permission:add"), permHandler.Create)
				prm.PUT("/:id", perm("system:permission:update"), permHandler.Update)
				prm.DELETE("/:id", perm("system:permission:delete"), permHandler.Delete)
			}
		}
	}
}
