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

// PermissionHandler 权限管理控制器
type PermissionHandler struct {
	permSvc *service.PermissionService
}

// NewPermissionHandler 创建 PermissionHandler 实例
func NewPermissionHandler(svc *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permSvc: svc}
}

// List 获取所有权限
// GET /api/admin/permission
func (h *PermissionHandler) List(c *gin.Context) {
	perms, err := h.permSvc.GetAllPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData(perms))
}

// GetByID 获取单个权限
// GET /api/admin/permission/:id
func (h *PermissionHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "无效的权限ID"))
		return
	}

	perm, err := h.permSvc.GetPermission(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPermissionNotFound) {
			c.JSON(http.StatusNotFound, common.Fail(404, "权限不存在"))
			return
		}
		c.JSON(http.StatusInternalServerError, common.Fail(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData(perm))
}

// Create 创建权限
// POST /api/admin/permission
func (h *PermissionHandler) Create(c *gin.Context) {
	var req dto.PermissionCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if err := common.ValidateRequired(req.PermissionName, "权限名称"); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateRequired(req.PermissionKey, "权限标识"); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateLength(req.PermissionName, "权限名称", 1, model.MaxLenPermName); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateLength(req.PermissionKey, "权限标识", 1, model.MaxLenPermKey); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if err := common.ValidateKeyFormat(req.PermissionKey, "权限标识"); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
		return
	}
	if req.Path != "" {
		if err := common.ValidateLength(req.Path, "权限路径", 1, model.MaxLenPermPath); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}
	if req.Method != "" {
		if err := common.ValidateLength(req.Method, "HTTP方法", 1, model.MaxLenPermMethod); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
		if err := common.ValidateMethod(req.Method); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}
	if req.Type != 1 && req.Type != 2 && req.Type != 3 {
		c.JSON(http.StatusBadRequest, common.Fail(400, "权限类型无效，应为1(菜单)、2(按钮)或3(API)"))
		return
	}
	if req.SortOrder < 0 || req.SortOrder > 9999 {
		c.JSON(http.StatusBadRequest, common.Fail(400, "排序值必须在0-9999之间"))
		return
	}

	perm, err := h.permSvc.CreatePermission(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "已存在") {
			c.JSON(http.StatusConflict, common.Fail(409, err.Error()))
			return
		}
		pkg.Logger.Error("CreatePermission failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, common.Fail(400, "创建权限失败"))
		return
	}
	c.JSON(http.StatusCreated, common.OkData(perm))
}

// Update 更新权限
// PUT /api/admin/permission/:id
func (h *PermissionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "无效的权限ID"))
		return
	}

	var req dto.PermissionUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if req.PermissionName != nil {
		if err := common.ValidateLength(*req.PermissionName, "权限名称", 1, model.MaxLenPermName); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}
	if req.PermissionKey != nil {
		if err := common.ValidateLength(*req.PermissionKey, "权限标识", 1, model.MaxLenPermKey); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
		if err := common.ValidateKeyFormat(*req.PermissionKey, "权限标识"); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}
	if req.Path != nil && *req.Path != "" {
		if err := common.ValidateLength(*req.Path, "权限路径", 1, model.MaxLenPermPath); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}
	if req.Method != nil && *req.Method != "" {
		if err := common.ValidateLength(*req.Method, "HTTP方法", 1, model.MaxLenPermMethod); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
		if err := common.ValidateMethod(*req.Method); err != nil {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
	}
	if req.Type != nil && (*req.Type != 1 && *req.Type != 2 && *req.Type != 3) {
		c.JSON(http.StatusBadRequest, common.Fail(400, "权限类型无效，应为1(菜单)、2(按钮)或3(API)"))
		return
	}
	if req.SortOrder != nil && (*req.SortOrder < 0 || *req.SortOrder > 9999) {
		c.JSON(http.StatusBadRequest, common.Fail(400, "排序值必须在0-9999之间"))
		return
	}

	if err := h.permSvc.UpdatePermission(c.Request.Context(), id, &req); err != nil {
		if errors.Is(err, service.ErrPermissionNotFound) {
			c.JSON(http.StatusNotFound, common.Fail(404, "权限不存在"))
			return
		}
		if strings.Contains(err.Error(), "已存在") {
			c.JSON(http.StatusConflict, common.Fail(409, err.Error()))
			return
		}
		pkg.Logger.Error("UpdatePermission failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Fail(500, "更新权限失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData(nil))
}

// Delete 删除权限
// DELETE /api/admin/permission/:id
func (h *PermissionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(400, "无效的权限ID"))
		return
	}

	if err := h.permSvc.DeletePermission(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "子权限") {
			c.JSON(http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, common.Fail(500, "删除失败"))
		return
	}
	c.JSON(http.StatusOK, common.OkData("删除成功"))
}
