package dto

// RoleCreateReq 创建角色请求
type RoleCreateReq struct {
	RoleName    string `json:"roleName"`
	RoleKey     string `json:"roleKey"`
	Description string `json:"description"`
}

// RoleUpdateReq 更新角色请求
type RoleUpdateReq struct {
	RoleName    *string `json:"roleName"`
	RoleKey     *string `json:"roleKey"`
	Description *string `json:"description"`
	Status      *int8   `json:"status"`
}

// AssignPermissionsReq 分配权限请求
type AssignPermissionsReq struct {
	PermissionIDs []int64 `json:"permissionIds"`
}
