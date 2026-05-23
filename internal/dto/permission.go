package dto

// PermissionCreateReq 创建权限请求
type PermissionCreateReq struct {
	PermissionName string `json:"permissionName"`
	PermissionKey  string `json:"permissionKey"`
	Type           int8   `json:"type"`
	ParentID       int64  `json:"parentId"`
	Path           string `json:"path"`
	Method         string `json:"method"`
	SortOrder      int    `json:"sortOrder"`
}

// PermissionUpdateReq 更新权限请求
type PermissionUpdateReq struct {
	PermissionName *string `json:"permissionName"`
	PermissionKey  *string `json:"permissionKey"`
	Type           *int8   `json:"type"`
	ParentID       *int64  `json:"parentId"`
	Path           *string `json:"path"`
	Method         *string `json:"method"`
	SortOrder      *int    `json:"sortOrder"`
	Status         *int8   `json:"status"`
}
