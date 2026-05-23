package handler

import (
	"errors"
	"net/http"
	"shopmall/common"
	"shopmall/internal/dto"
	"shopmall/internal/model"
	"shopmall/internal/service"
	"shopmall/pkg"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RoleHandler 角色管理控制器
type RoleHandler struct {
	roleSvc *service.RoleService
}

// NewRoleHandler 创建 RoleHandler 实例
func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{roleSvc: svc}
}

// List 获取所有角色
// GET /api/admin/role
func (h *RoleHandler) List(c *gin.Context) {
	roles, err := h.roleSvc.GetAllRoles(c.Request.Context())
	if err != nil {
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData(roles))
}

// GetByID 获取单个角色
// GET /api/admin/role/:id
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "无效的角色ID"))
		return
	}

	role, err := h.roleSvc.GetRole(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, common.Fail(404, "角色不存在"))
			return
		}
		c.JSON(http.StatusInternalServerError, common.Fail(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData(role))
}

// Create 创建角色
// POST /api/admin/role
func (h *RoleHandler) Create(c *gin.Context) {
	var req dto.RoleCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if err := common.ValidateRequired(req.RoleName, "角色名称"); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateRequired(req.RoleKey, "角色标识"); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateLength(req.RoleName, "角色名称", 1, model.MaxLenRoleName); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateLength(req.RoleKey, "角色标识", 1, model.MaxLenRoleKey); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateKeyFormat(req.RoleKey, "角色标识"); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateLength(req.Description, "角色描述", 0, model.MaxLenRoleDesc); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}

	role, err := h.roleSvc.CreateRole(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "已存在") || strings.Contains(err.Error(), "Duplicate") {
			c.JSON(http.StatusConflict, common.Fail(409, "角色标识已存在"))
			return
		}
		pkg.Logger.Error("CreateRole failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Fail(500, "创建角色失败"))
		return
	}
	c.JSON(http.StatusCreated, common.OkData(role))
}

// Update 更新角色
// PUT /api/admin/role/:id
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "无效的角色ID"))
		return
	}

	var req dto.RoleUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if req.RoleName != nil {
		if err := common.ValidateLength(*req.RoleName, "角色名称", 1, model.MaxLenRoleName); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}
	if req.RoleKey != nil {
		if err := common.ValidateLength(*req.RoleKey, "角色标识", 1, model.MaxLenRoleKey); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
		if err := common.ValidateKeyFormat(*req.RoleKey, "角色标识"); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}
	if req.Description != nil {
		if err := common.ValidateLength(*req.Description, "角色描述", 0, model.MaxLenRoleDesc); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}

	if err := h.roleSvc.UpdateRole(c.Request.Context(), id, &req); err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, common.Fail(404, "角色不存在"))
			return
		}
		if strings.Contains(err.Error(), "已存在") || strings.Contains(err.Error(), "Duplicate") {
			c.JSON(http.StatusConflict, common.Fail(409, err.Error()))
			return
		}
		pkg.Logger.Error("UpdateRole failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Fail(500, "更新角色失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData(nil))
}

// Delete 删除角色
// DELETE /api/admin/role/:id
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "无效的角色ID"))
		return
	}

	if err := h.roleSvc.DeleteRole(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(500, "删除失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData("删除成功"))
}

// GetPermissionIDs 获取角色权限ID列表
// GET /api/admin/role/:id/permissions
func (h *RoleHandler) GetPermissionIDs(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "无效的角色ID"))
		return
	}

	ids, err := h.roleSvc.GetRolePermissionIDs(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData(ids))
}

// AssignPermissions 分配权限给角色
// PUT /api/admin/role/:id/permissions
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "无效的角色ID"))
		return
	}

	var req dto.AssignPermissionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if len(req.PermissionIDs) > 200 {
		c.JSON(http.StatusBadRequest, common.Fail(400, "单次分配权限数量不能超过200个"))
		return
	}

	if err := h.roleSvc.AssignPermissions(c.Request.Context(), id, &req); err != nil {
		pkg.Logger.Error("AssignPermissions failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, common.Fail(400, "分配权限失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData("分配成功"))
}
