package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"shopmall/internal/dto"
	"shopmall/internal/model"
	"shopmall/internal/repository"
)

var (
	ErrPermissionNotFound = errors.New("权限不存在")
)

// PermissionService 权限服务
type PermissionService struct {
	repo *repository.PermissionRepo
}

// NewPermissionService 创建 PermissionService 实例
func NewPermissionService(repo *repository.PermissionRepo) *PermissionService {
	return &PermissionService{repo: repo}
}

// GetAllPermissions 获取所有权限
func (s *PermissionService) GetAllPermissions(ctx context.Context) ([]*model.SysPermission, error) {
	return s.repo.ListAll(ctx)
}

// GetPermission 获取单个权限
func (s *PermissionService) GetPermission(ctx context.Context, id int64) (*model.SysPermission, error) {
	return s.repo.GetByID(ctx, id)
}

// CreatePermission 创建权限
func (s *PermissionService) CreatePermission(ctx context.Context, req *dto.PermissionCreateReq) (*model.SysPermission, error) {
	perm := &model.SysPermission{
		PermissionName: req.PermissionName,
		PermissionKey:  req.PermissionKey,
		Type:           model.PermissionType(req.Type),
		ParentID:       req.ParentID,
		Path:           req.Path,
		Method:         req.Method,
		SortOrder:      req.SortOrder,
		Status:         1,
	}
	if err := s.repo.Create(ctx, perm); err != nil {
		return nil, err
	}
	return perm, nil
}

// UpdatePermission 更新权限（支持部分更新）
func (s *PermissionService) UpdatePermission(ctx context.Context, id int64, req *dto.PermissionUpdateReq) error {
	perm, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPermissionNotFound
		}
		return fmt.Errorf("查询权限失败: %w", err)
	}
	if req.PermissionName != nil {
		perm.PermissionName = *req.PermissionName
	}
	if req.PermissionKey != nil {
		perm.PermissionKey = *req.PermissionKey
	}
	if req.Type != nil {
		perm.Type = model.PermissionType(*req.Type)
	}
	if req.ParentID != nil {
		perm.ParentID = *req.ParentID
	}
	if req.Path != nil {
		perm.Path = *req.Path
	}
	if req.Method != nil {
		perm.Method = *req.Method
	}
	if req.SortOrder != nil {
		perm.SortOrder = *req.SortOrder
	}
	if req.Status != nil {
		perm.Status = *req.Status
	}
	return s.repo.Update(ctx, perm)
}

// DeletePermission 删除权限（同时清理角色-权限关联）
func (s *PermissionService) DeletePermission(ctx context.Context, id int64) error {
	return s.repo.DeleteWithRoleRefs(ctx, id)
}
