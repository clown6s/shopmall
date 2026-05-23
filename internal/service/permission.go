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
	perm, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPermissionNotFound
		}
		return nil, fmt.Errorf("查询权限失败: %w", err)
	}
	return perm, nil
}

// CreatePermission 创建权限
func (s *PermissionService) CreatePermission(ctx context.Context, req *dto.PermissionCreateReq) (*model.SysPermission, error) {
	// 校验 PermissionKey 唯一性
	existing, err := s.repo.GetByKey(ctx, req.PermissionKey)
	if err == nil && existing != nil {
		return nil, errors.New("权限标识已存在")
	}

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
		if isDuplicateKeyError(err) {
			return nil, errors.New("权限标识已存在")
		}
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
	if req.PermissionKey != nil && *req.PermissionKey != perm.PermissionKey {
		// 校验新 PermissionKey 唯一性
		existing, err := s.repo.GetByKey(ctx, *req.PermissionKey)
		if err == nil && existing != nil && existing.ID != id {
			return errors.New("权限标识已存在")
		}
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
	if err := s.repo.Update(ctx, perm); err != nil {
		if isDuplicateKeyError(err) {
			return errors.New("权限标识已存在")
		}
		return err
	}
	return nil
}

// DeletePermission 删除权限（同时清理角色-权限关联）
func (s *PermissionService) DeletePermission(ctx context.Context, id int64) error {
	// 检查是否存在子权限
	hasChildren, err := s.repo.HasChildPermissions(ctx, id)
	if err != nil {
		return fmt.Errorf("检查子权限失败: %w", err)
	}
	if hasChildren {
		return errors.New("该权限下存在子权限，请先删除子权限")
	}
	return s.repo.DeleteWithRoleRefs(ctx, id)
}
