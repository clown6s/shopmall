package middleware

import (
	"net/http"
	"shopmall/internal/repository"

	"github.com/gin-gonic/gin"
)

// RequireAdmin 返回管理员权限中间件
// 仅允许 super_admin 或 admin 角色访问
func RequireAdmin(empRepo *repository.EmployeeRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		empID := GetEmployeeID(c)
		if empID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
			c.Abort()
			return
		}

		emp, err := empRepo.GetByIDWithoutPassword(c.Request.Context(), empID)
		if err != nil || emp == nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权执行此操作"})
			c.Abort()
			return
		}

		if emp.Role != "super_admin" && emp.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权执行此操作"})
			c.Abort()
			return
		}

		c.Next()
	}
}
