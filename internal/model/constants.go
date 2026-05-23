package model

// 分页默认值
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// 密码默认值
const (
	PasswordMinLen = 8
	PasswordMaxLen = 64
)

// 字段长度上限
const (
	MaxLenRealName     = 50
	MaxLenDepartment   = 50
	MaxLenPosition     = 50
	MaxLenRemark       = 200
	MaxLenDeptName     = 50
	MaxLenCategoryName = 50
	MaxLenBrandName    = 50
	MaxLenBrandDesc    = 500
	MaxLenBrandLogo    = 500
	MaxLenEmail        = 100
)

// RBAC 管理字段长度上限
const (
	MaxLenRoleName   = 50
	MaxLenRoleKey    = 50
	MaxLenRoleDesc   = 200
	MaxLenPermName   = 50
	MaxLenPermKey    = 100
	MaxLenPermPath   = 200
	MaxLenPermMethod = 10
)
