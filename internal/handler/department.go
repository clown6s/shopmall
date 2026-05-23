package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"shopmall/common"
	"shopmall/internal/model"
	"shopmall/internal/repository"
	"shopmall/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"shopmall/pkg"
)

// DepartmentHandler 部门 Handler
type DepartmentHandler struct {
	DepartmentService *service.DepartmentService
}

func NewDepartmentHandler(svc *service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{DepartmentService: svc}
}

// GetAll 获取部门列表（支持分页）
// GET /api/department?page=1&pageSize=10&deptName=xxx
func (h *DepartmentHandler) GetAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}
	deptName := c.Query("deptName")

	list, total, err := h.DepartmentService.GetAllWithPage(ctx, page, pageSize, deptName)
	if err != nil {
		pkg.Logger.Error("GetAll departments failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(map[string]interface{}{
		"list":  list,
		"total": total,
	}))
}

// GetAllSimple 获取所有部门（无分页）
// GET /api/department/all
func (h *DepartmentHandler) GetAllSimple(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	list, err := h.DepartmentService.GetAll(ctx)
	if err != nil {
		pkg.Logger.Error("GetAllSimple departments failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}
	common.WriteJSON(c, http.StatusOK, common.OkData(list))
}

// GetByID 获取单个部门
// GET /api/department/:deptId
func (h *DepartmentHandler) GetByID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(c.Param("deptId"), 10, 64)
	if err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "参数错误：ID无效"))
		return
	}

	dept, err := h.DepartmentService.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "部门不存在"))
			return
		}
		pkg.Logger.Error("GetByID department failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}
	common.WriteJSON(c, http.StatusOK, common.OkData(dept))
}

// Create 新增部门
// POST /api/department
func (h *DepartmentHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	var req struct {
		DeptName string `json:"deptName" binding:"required"`
		DeptSort int    `json:"deptSort"`
		Remark   string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if len(req.DeptName) > model.MaxLenDeptName {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "部门名称长度不能超过50个字符"))
		return
	}
	if req.DeptSort < 0 || req.DeptSort > 9999 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "排序值必须在0-9999之间"))
		return
	}
	if len(req.Remark) > model.MaxLenRemark {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "备注长度不能超过200个字符"))
		return
	}

	dept := &model.SysDepartment{
		DeptName: req.DeptName,
		DeptSort: req.DeptSort,
		Remark:   req.Remark,
	}

	id, err := h.DepartmentService.Create(ctx, dept)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParam) {
			common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "部门名称不能为空"))
			return
		}
		pkg.Logger.Error("Create department failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusCreated, common.OkData(map[string]int64{"deptId": id}))
}

// Update 更新部门
// PUT /api/department/:deptId
func (h *DepartmentHandler) Update(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(c.Param("deptId"), 10, 64)
	if err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "参数错误：ID无效"))
		return
	}

	var req struct {
		DeptName string `json:"deptName" binding:"required"`
		DeptSort int    `json:"deptSort"`
		Remark   string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if len(req.DeptName) > model.MaxLenDeptName {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "部门名称长度不能超过50个字符"))
		return
	}
	if req.DeptSort < 0 || req.DeptSort > 9999 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "排序值必须在0-9999之间"))
		return
	}
	if len(req.Remark) > model.MaxLenRemark {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "备注长度不能超过200个字符"))
		return
	}

	dept := &model.SysDepartment{
		DeptName: req.DeptName,
		DeptSort: req.DeptSort,
		Remark:   req.Remark,
	}

	if err := h.DepartmentService.Update(ctx, id, dept); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "部门不存在"))
			return
		}
		if errors.Is(err, service.ErrInvalidParam) {
			common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "部门名称不能为空"))
			return
		}
		pkg.Logger.Error("Update department failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(nil))
}

// Delete 删除部门
// DELETE /api/department/:deptId
func (h *DepartmentHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(c.Param("deptId"), 10, 64)
	if err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "参数错误：ID无效"))
		return
	}

	if err := h.DepartmentService.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "部门不存在"))
			return
		}
		pkg.Logger.Error("Delete department failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(nil))
}
