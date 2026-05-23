package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"shopmall/common"
	"shopmall/internal/dto"
	"shopmall/internal/model"
	"shopmall/internal/repository"
	"shopmall/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"shopmall/pkg"
)

// ProductHandler 商品 Handler
type ProductHandler struct {
	ProductService *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{ProductService: svc}
}

// CategoryTree 获取分类级联树
// GET /api/product/category
func (h *ProductHandler) CategoryTree(c *gin.Context) {
	gzipData, etag := h.ProductService.CategoryTree()

	// 304 Not Modified
	if clientETag := c.GetHeader("If-None-Match"); clientETag != "" && strings.Trim(clientETag, `"`) == strings.Trim(etag, `"`) {
		c.Header("ETag", etag)
		c.Header("Cache-Control", "no-cache")
		c.Status(http.StatusNotModified)
		return
	}

	c.Header("Content-Encoding", "gzip")
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("ETag", etag)
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)
	c.Writer.Write(gzipData)
}

// CategoryAdd 添加分类
// POST /api/product/category
func (h *ProductHandler) CategoryAdd(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	var req dto.CategoryAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if req.Name == "" {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "分类名称不能为空"))
		return
	}
	if len(req.Name) > model.MaxLenCategoryName {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "分类名称长度不能超过50个字符"))
		return
	}
	if req.CatLevel <= 0 || req.CatLevel > 3 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "分类层级无效"))
		return
	}
	if req.Sort < 0 || req.Sort > 9999 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "排序值必须在0-9999之间"))
		return
	}
	if req.ShowStatus != 0 && req.ShowStatus != 1 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "显示状态值无效"))
		return
	}
	if len(req.Icon) > 500 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "分类图标长度不能超过500个字符"))
		return
	}

	id, err := h.ProductService.AddCategory(ctx, &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParam) {
			common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "参数无效"))
			return
		}
		pkg.Logger.Error("CategoryAdd failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusCreated, common.OkData(map[string]int64{"catId": id}))
}

// CategoryEdit 编辑分类
// PUT /api/product/category
func (h *ProductHandler) CategoryEdit(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	var req dto.CategoryEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if len(req.Name) > model.MaxLenCategoryName {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "分类名称长度不能超过50个字符"))
		return
	}
	if req.Sort < 0 || req.Sort > 9999 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "排序值必须在0-9999之间"))
		return
	}
	if req.ShowStatus != 0 && req.ShowStatus != 1 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "显示状态值无效"))
		return
	}

	if err := h.ProductService.EditCategory(ctx, &req); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "分类不存在"))
			return
		}
		if errors.Is(err, service.ErrInvalidParam) {
			common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "分类ID和名称不能为空"))
			return
		}
		pkg.Logger.Error("CategoryEdit failed", zap.Int64("catId", req.CatID), zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(nil))
}

// CategoryDelete 删除分类（逻辑删除）
// DELETE /api/product/category/:catId
func (h *ProductHandler) CategoryDelete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(c.Param("catId"), 10, 64)
	if err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "参数错误：ID无效"))
		return
	}

	categoryCount, brandCount, err := h.ProductService.DeleteCategory(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "分类不存在"))
			return
		}
		pkg.Logger.Error("CategoryDelete failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(map[string]int64{
		"categoryCount": categoryCount,
		"brandCount":    brandCount,
	}))
}

// ==================== 品牌相关 ====================

// GetBrands 获取分类关联品牌列表
// GET /api/product/brands?catId=xxx
func (h *ProductHandler) GetBrands(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	catIdStr := c.Query("catId")
	if catIdStr == "" {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "分类ID不能为空"))
		return
	}
	catID, err := strconv.ParseInt(catIdStr, 10, 64)
	if err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "invalid catId"))
		return
	}

	brands, err := h.ProductService.GetBrandsByCatID(ctx, catID)
	if err != nil {
		pkg.Logger.Error("GetBrands failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(brands))
}

// AddBrand 添加品牌
// POST /api/product/brand
func (h *ProductHandler) AddBrand(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	var req struct {
		CatId       int64  `json:"catId"`
		Name        string `json:"name" binding:"required"`
		Logo        string `json:"logo"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if len(req.Name) > model.MaxLenBrandName {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "品牌名称长度不能超过50个字符"))
		return
	}
	if len(req.Description) > model.MaxLenBrandDesc {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "品牌描述长度不能超过500个字符"))
		return
	}
	if len(req.Logo) > model.MaxLenBrandLogo {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "品牌Logo长度不能超过500个字符"))
		return
	}
	if req.CatId <= 0 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "关联分类ID无效"))
		return
	}

	brand := &model.Brand{
		CatId:       req.CatId,
		Name:        req.Name,
		Logo:        req.Logo,
		Description: req.Description,
	}

	id, err := h.ProductService.AddBrand(ctx, brand)
	if err != nil {
		pkg.Logger.Error("AddBrand failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusCreated, common.OkData(id))
}

// UpdateBrand 更新品牌
// PUT /api/product/brand/:brandId
func (h *ProductHandler) UpdateBrand(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	brandID, err := strconv.ParseInt(c.Param("brandId"), 10, 64)
	if err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "invalid brandId"))
		return
	}

	var req struct {
		Name        string `json:"name"`
		Logo        string `json:"logo"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if req.Name == "" {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "品牌名称不能为空"))
		return
	}
	if len(req.Name) > model.MaxLenBrandName {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "品牌名称长度不能超过50个字符"))
		return
	}
	if len(req.Description) > model.MaxLenBrandDesc {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "品牌描述长度不能超过500个字符"))
		return
	}
	if len(req.Logo) > model.MaxLenBrandLogo {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "品牌Logo长度不能超过500个字符"))
		return
	}

	brand := &model.Brand{
		Name:        req.Name,
		Logo:        req.Logo,
		Description: req.Description,
	}

	if err := h.ProductService.UpdateBrand(ctx, brandID, brand); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "品牌不存在"))
			return
		}
		pkg.Logger.Error("UpdateBrand failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(nil))
}

// DeleteBrand 删除品牌（物理删除）
// DELETE /api/product/brand/:brandId
func (h *ProductHandler) DeleteBrand(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	brandID, err := strconv.ParseInt(c.Param("brandId"), 10, 64)
	if err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "参数错误：品牌ID无效"))
		return
	}

	if err := h.ProductService.DeleteBrand(ctx, brandID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "品牌不存在"))
			return
		}
		pkg.Logger.Error("DeleteBrand failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "服务器内部错误"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(nil))
}
