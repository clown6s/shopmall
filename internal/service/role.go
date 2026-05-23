package service

import "context"

// RoleService 角色服务（桩实现，后续补充完整逻辑）
type RoleService struct{}

// NewRoleSvc 创建 RoleService 实例
func NewRoleSvc() *RoleService {
	return &RoleService{}
}

// GetUserMenus 获取用户角色对应的菜单权限
// 返回菜单树和按钮权限 key 列表
func (s *RoleService) GetUserMenus(_ context.Context, role string) ([]interface{}, []string, error) {
	return []interface{}{}, []string{}, nil
}
