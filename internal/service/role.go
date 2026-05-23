package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"shopmall/internal/dto"
	"shopmall/internal/model"
	"shopmall/internal/repository"
	"shopmall/pkg"

	"go.uber.org/zap"
)

var (
	ErrRoleNotFound = errors.New("角色不存在")
)

// RoleService 角色服务
type RoleService struct {
	repo           *repository.RoleRepo
	permissionRepo *repository.PermissionRepo
	employeeRepo   *repository.EmployeeRepo
}

// NewRoleSvc 创建 RoleService 实例
func NewRoleSvc(repo *repository.RoleRepo, permRepo *repository.PermissionRepo, empRepo *repository.EmployeeRepo) *RoleService {
	return &RoleService{repo: repo, permissionRepo: permRepo, employeeRepo: empRepo}
}

// GetAllRoles 获取所有角色
func (s *RoleService) GetAllRoles(ctx context.Context) ([]*model.SysRole, error) {
	return s.repo.ListAll(ctx)
}

// GetRole 获取单个角色
func (s *RoleService) GetRole(ctx context.Context, id int64) (*model.SysRole, error) {
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return role, nil
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(ctx context.Context, req *dto.RoleCreateReq) (*model.SysRole, error) {
	// 校验 RoleName 唯一性
	existing, err := s.repo.GetByRoleName(ctx, req.RoleName)
	if err == nil && existing != nil {
		return nil, errors.New("角色名称已存在")
	}
	// 校验 RoleKey 唯一性
	existing, err = s.repo.GetByRoleKey(ctx, req.RoleKey)
	if err == nil && existing != nil {
		return nil, errors.New("角色标识已存在")
	}

	role := &model.SysRole{
		RoleName:    req.RoleName,
		RoleKey:     req.RoleKey,
		Description: req.Description,
		Status:      1,
	}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRole 更新角色（支持部分更新）
func (s *RoleService) UpdateRole(ctx context.Context, id int64, req *dto.RoleUpdateReq) error {
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("查询角色失败: %w", err)
	}
	if req.RoleKey != nil {
		return errors.New("角色标识不允许修改")
	}
	if req.RoleName != nil {
		role.RoleName = *req.RoleName
		// 校验 RoleName 唯一性（应用层检查 + DB UNIQUE 约束兜底）
		if existing, err := s.repo.GetByRoleName(ctx, *req.RoleName); err == nil && existing != nil && existing.ID != id {
			return errors.New("角色名称已存在")
		}
	}
	// role_key 不允许修改，避免与员工表的 role 字段关联断裂
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Status != nil {
		role.Status = *req.Status
	}
	return s.repo.Update(ctx, role)
}

// DeleteRole 删除角色（事务性：同时删除角色-权限关联）
func (s *RoleService) DeleteRole(ctx context.Context, id int64) error {
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("查询角色失败: %w", err)
	}
	count, err := s.employeeRepo.CountByRole(ctx, role.RoleKey)
	if err != nil {
		return fmt.Errorf("查询关联员工失败: %w", err)
	}
	if count > 0 {
		return errors.New("该角色下存在关联员工，请先转移或删除员工")
	}
	return s.repo.DeleteWithPermissions(ctx, id)
}

// GetRolePermissionIDs 获取角色权限ID列表
func (s *RoleService) GetRolePermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	return s.repo.GetPermissionIDs(ctx, roleID)
}

// AssignPermissions 分配权限给角色
func (s *RoleService) AssignPermissions(ctx context.Context, roleID int64, req *dto.AssignPermissionsReq) error {
	if len(req.PermissionIDs) > 0 {
		count, err := s.permissionRepo.CountByIDs(ctx, req.PermissionIDs)
		if err != nil {
			return fmt.Errorf("验证权限ID失败: %w", err)
		}
		if int(count) != len(req.PermissionIDs) {
			return errors.New("部分权限ID不存在")
		}
	}
	return s.repo.SetPermissions(ctx, roleID, req.PermissionIDs)
}

// GetUserMenus 根据用户角色获取菜单路径 + 按钮权限keys
func (s *RoleService) GetUserMenus(ctx context.Context, role string) ([]string, []string, error) {
	if role == "" {
		return []string{}, []string{}, nil
	}

	// 1. 根据 role 字符串查 sys_role
	roleEntity, err := s.repo.GetByRoleKey(ctx, role)
	if err != nil {
		pkg.Logger.Warn("GetUserMenus: role not found", zap.String("role", role), zap.Error(err))
		return []string{}, []string{}, nil
	}

	// 2. 查该角色的菜单权限（type=1，含子权限的父菜单）
	perms, err := s.permissionRepo.GetMenuPermissionsByRoleID(ctx, roleEntity.ID)
	if err != nil {
		return nil, nil, err
	}

	// 3. 提取菜单路径
	paths := make([]string, 0, len(perms))
	for _, p := range perms {
		if p.Path != "" {
			paths = append(paths, p.Path)
		}
	}

	// 4. 查该角色的按钮权限keys（type=2）
	btnKeys, err := s.permissionRepo.GetButtonPermissionKeysByRoleID(ctx, roleEntity.ID)
	if err != nil {
		return nil, nil, err
	}
	return paths, btnKeys, nil
}
