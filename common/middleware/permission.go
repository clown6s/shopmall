package middleware

import (
	"net/http"
	"shopmall/internal/repository"

	"github.com/gin-gonic/gin"
)

// RequirePermission 返回 API 权限校验中间件
// 通过 employee → role → sys_role_permission → sys_permission 链路查询用户的 permission_key
// super_admin 角色直接放行
func RequirePermission(empRepo *repository.EmployeeRepo, roleRepo *repository.RoleRepo, permRepo *repository.PermissionRepo, keys ...string) gin.HandlerFunc {
	requiredSet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		requiredSet[k] = struct{}{}
	}

	return func(c *gin.Context) {
		empID := GetEmployeeID(c)
		if empID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
			c.Abort()
			return
		}

		ctx := c.Request.Context()

		// 1. 查员工信息，拿 role（role_key 字符串）
		emp, err := empRepo.GetByIDWithoutPassword(ctx, empID)
		if err != nil || emp == nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权执行此操作"})
			c.Abort()
			return
		}

		// 2. super_admin 直接放行
		if emp.Role == "super_admin" {
			c.Next()
			return
		}

		// 3. 通过 role_key 查 role_id
		role, err := roleRepo.GetByRoleKey(ctx, emp.Role)
		if err != nil || role == nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权执行此操作"})
			c.Abort()
			return
		}

		// 4. 查该角色的所有 permission_key（type=2 按钮 + type=3 API）
		permKeys, err := permRepo.GetButtonPermissionKeysByRoleID(ctx, role.ID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权执行此操作"})
			c.Abort()
			return
		}

		// 5. 匹配
		for _, k := range permKeys {
			if _, ok := requiredSet[k]; ok {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权执行此操作"})
		c.Abort()
	}
}
