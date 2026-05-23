package model

import "time"

// SysRole 角色表
type SysRole struct {
	ID          int64     `json:"id"`
	RoleName    string    `json:"roleName"`
	RoleKey     string    `json:"roleKey"`
	Description string    `json:"description"`
	Status      int8      `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (SysRole) TableName() string {
	return "sys_role"
}

// SysRolePermission 角色-权限关联表
type SysRolePermission struct {
	ID           int64 `json:"id"`
	RoleID       int64 `json:"roleId"`
	PermissionID int64 `json:"permissionId"`
}

func (SysRolePermission) TableName() string {
	return "sys_role_permission"
}
