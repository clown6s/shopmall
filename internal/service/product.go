package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"shopmall/internal/dto"
	"shopmall/internal/model"
	"shopmall/internal/repository"
	"sync"
	"sync/atomic"
	"time"
)

// ProductService 商品业务逻辑
type ProductService struct {
	productRepo    *repository.ProductRepo
	categoryGzip   atomic.Value // []byte
	refreshMu      sync.Mutex
	refreshTimer   *time.Timer
	refreshPending atomic.Bool
	closed         atomic.Bool
}

var jsonPrefix = []byte(`{"code":200,"message":"success","data":`)

func NewProductSvc(repo *repository.ProductRepo) *ProductService {
	s := &ProductService{productRepo: repo}
	s.categoryGzip.Store(defaultGzipData())
	return s
}

func defaultGzipData() []byte {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return []byte(`{"code":200,"message":"success","data":[]}`)
	}
	if _, err := gz.Write([]byte(`{"code":200,"message":"success","data":[]}`)); err != nil {
		return []byte(`{"code":200,"message":"success","data":[]}`)
	}
	if err := gz.Close(); err != nil {
		return []byte(`{"code":200,"message":"success","data":[]}`)
	}
	return buf.Bytes()
}

// InitCategoryCache 初始化缓存（启动时调用）
func (s *ProductService) InitCategoryCache(ctx context.Context) error {
	list, err := s.productRepo.GetAllCategories(ctx)
	if err != nil {
		return err
	}

	tree := BuildCategoryTree(list)

	dataJSON, err := json.Marshal(tree)
	if err != nil {
		return err
	}

	fullJSON := make([]byte, 0, len(jsonPrefix)+len(dataJSON)+1)
	fullJSON = append(fullJSON, jsonPrefix...)
	fullJSON = append(fullJSON, dataJSON...)
	fullJSON = append(fullJSON, '}')

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return err
	}
	if _, err = gz.Write(fullJSON); err != nil {
		gz.Close()
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}

	s.categoryGzip.Store(buf.Bytes())
	return nil
}

// CategoryTree 查询分类树（gzip缓存）
func (s *ProductService) CategoryTree() []byte {
	return s.categoryGzip.Load().([]byte)
}

// AddCategory 添加分类
func (s *ProductService) AddCategory(ctx context.Context, req *dto.CategoryAddRequest) (int64, error) {
	if req.Name == "" {
		return 0, ErrInvalidParam
	}
	if req.CatLevel <= 0 || req.CatLevel > 3 {
		return 0, ErrInvalidParam
	}

	cat := &model.Category{
		Name:       req.Name,
		ParentCid:  req.ParentCid,
		CatLevel:   req.CatLevel,
		ShowStatus: req.ShowStatus,
		Sort:       req.Sort,
	}
	if req.Icon != "" {
		cat.Icon = &req.Icon
	}

	id, err := s.productRepo.AddCategory(ctx, cat)
	if err != nil {
		return 0, err
	}

	s.refreshCacheDebounced()
	return id, nil
}

// EditCategory 编辑分类
func (s *ProductService) EditCategory(ctx context.Context, req *dto.CategoryEditRequest) error {
	if req.CatID == 0 {
		return ErrInvalidParam
	}
	if req.Name == "" {
		return ErrInvalidParam
	}

	if err := s.productRepo.EditCategory(ctx, req.CatID, req.Name, req.Sort, req.ShowStatus); err != nil {
		return err
	}

	s.refreshCacheDebounced()
	return nil
}

// DeleteCategory 删除分类
func (s *ProductService) DeleteCategory(ctx context.Context, catID int64) (int64, int64, error) {
	catRows, brandRows, err := s.productRepo.DeleteCategory(ctx, catID)
	if err != nil {
		return 0, 0, err
	}
	s.refreshCacheDebounced()
	return catRows, brandRows, nil
}

// GetBrandsByCatID 获取分类关联品牌列表
func (s *ProductService) GetBrandsByCatID(ctx context.Context, catID int64) ([]model.Brand, error) {
	return s.productRepo.GetBrandsByCatID(ctx, catID)
}

// AddBrand 添加品牌
func (s *ProductService) AddBrand(ctx context.Context, brand *model.Brand) (int64, error) {
	return s.productRepo.AddBrand(ctx, brand)
}

// UpdateBrand 更新品牌
func (s *ProductService) UpdateBrand(ctx context.Context, brandID int64, brand *model.Brand) error {
	return s.productRepo.UpdateBrand(ctx, brandID, brand)
}

// DeleteBrand 删除品牌
func (s *ProductService) DeleteBrand(ctx context.Context, brandID int64) error {
	return s.productRepo.DeleteBrand(ctx, brandID)
}

// refreshCacheDebounced 防抖刷新缓存
func (s *ProductService) refreshCacheDebounced() {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	if s.closed.Load() {
		return
	}
	if s.refreshTimer != nil {
		s.refreshTimer.Stop()
	}
	s.refreshTimer = time.AfterFunc(500*time.Millisecond, func() {
		if s.closed.Load() {
			return
		}
		go func() {
			_ = s.InitCategoryCache(context.Background())
		}()
	})
}

// BuildCategoryTree 构建分类级联树
func BuildCategoryTree(categories []model.Category) []*dto.CascaderNode {
	nodeMap := make(map[int64]*dto.CascaderNode, len(categories))
	roots := make([]*dto.CascaderNode, 0, len(categories)/3)

	for _, c := range categories {
		nodeMap[c.CatID] = &dto.CascaderNode{
			CatID:      c.CatID,
			Name:       c.Name,
			Sort:       c.Sort,
			CatLevel:   c.CatLevel,
			ShowStatus: c.ShowStatus,
			Children:   make([]*dto.CascaderNode, 0),
		}
	}

	for _, c := range categories {
		node := nodeMap[c.CatID]
		if c.ParentCid == 0 {
			roots = append(roots, node)
			continue
		}
		if parent, ok := nodeMap[c.ParentCid]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return roots
}

// Close 停止刷新定时器
func (s *ProductService) Close() {
	s.closed.Store(true)
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	if s.refreshTimer != nil {
		s.refreshTimer.Stop()
	}
}
