package dto

import (
	"shopmall/internal/model"
	"time"
)

// ===================== Login =====================

// LoginRequest 登录请求
type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// LoginResponse 登录响应（token 通过 httpOnly Cookie 传递）
type LoginResponse struct {
	EmployeeInfo *EmployeeInfo `json:"employeeInfo"`
}

// ===================== Employee Info =====================

// EmployeeInfo 员工信息（对外返回，不含密码）
type EmployeeInfo struct {
	ID         int64                `json:"employeeId"`
	RealName   string               `json:"realName"`
	Role       string               `json:"role"`
	Age        *int                 `json:"age,omitempty"`
	Phone      string               `json:"phone"`
	Email      string               `json:"email,omitempty"`
	Avatar     string               `json:"avatar,omitempty"`
	Gender     int8                 `json:"gender"`
	Department string               `json:"department,omitempty"`
	Position   string               `json:"position,omitempty"`
	WorkStatus model.WorkStatusType `json:"workStatus"`
	CanLogin   model.CanLoginType   `json:"canLogin"`
	CreateTime time.Time            `json:"createTime"`
}

// ===================== Employee List =====================

// EmployeeListRequest 员工列表查询请求
type EmployeeListRequest struct {
	RealName string `json:"realName"`
	Phone    string `json:"phone"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

// EmployeeListResponse 员工列表响应
type EmployeeListResponse struct {
	List     []*EmployeeInfo `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// ===================== Create Employee =====================

// CreateEmployeeRequest 添加员工请求
type CreateEmployeeRequest struct {
	RealName   string `json:"realName"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	Email      string `json:"email,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
	Gender     int8   `json:"gender"`
	Age        *int   `json:"age,omitempty"`
	Department string `json:"department,omitempty"`
	Position   string `json:"position,omitempty"`
	Role       string `json:"role,omitempty"`
	WorkStatus int8   `json:"workStatus"`
	CanLogin   int8   `json:"canLogin"`
	Remark     string `json:"remark,omitempty"`
}

// ===================== Update Employee =====================

// UpdateEmployeeRequest 更新员工信息请求（仅允许本人更新非敏感字段）
type UpdateEmployeeRequest struct {
	RealName string `json:"realName"`
	Phone    string `json:"phone,omitempty"`
	Email    string `json:"email,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Gender   int8   `json:"gender"`
	Age      *int   `json:"age,omitempty"`
	Remark   string `json:"remark,omitempty"`
}

// ===================== Password =====================

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// ResetPasswordRequest 管理员重置密码请求
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword"`
}
