package model

import (
	"time"
)

// WorkStatusType 在职状态
type WorkStatusType int8

const (
	WorkStatusDimission WorkStatusType = 0 // 离职
	WorkStatusOnJob     WorkStatusType = 1 // 在职
)

// CanLoginType 是否允许登录后台
type CanLoginType int8

const (
	CanLoginNo  CanLoginType = 0 // 否
	CanLoginYes CanLoginType = 1 // 是
)

// GenderType 性别
type GenderType int8

const (
	GenderUnknown GenderType = 0 // 未知
	GenderMale    GenderType = 1 // 男
	GenderFemale  GenderType = 2 // 女
	GenderOther   GenderType = 3 // 其他
)

// SysDepartment 部门表
type SysDepartment struct {
	DeptID     int64     `json:"deptId"`
	DeptName   string    `json:"deptName"`
	DeptSort   int       `json:"deptSort"`
	Status     int8      `json:"status"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
	Remark     string    `json:"remark"`
}

func (SysDepartment) TableName() string {
	return "sys_department"
}

// SysEmployee 员工表
type SysEmployee struct {
	EmployeeID int64          `json:"employeeId"`
	Password   *string        `json:"-"` // 加密后的密码，禁止序列化
	RealName   string         `json:"realName"`
	Avatar     *string        `json:"avatar"`
	Gender     int8           `json:"gender"` // 0-未知 1-男 2-女
	Age        *int           `json:"age"`
	Phone      string         `json:"phone"` // 手机号
	Email      string         `json:"email"`
	Department string         `json:"department"`
	Position   string         `json:"position"`
	Role       string         `json:"role"`
	WorkStatus WorkStatusType `json:"workStatus"`
	CanLogin   CanLoginType   `json:"canLogin"`
	CreateTime time.Time      `json:"createTime"`
	UpdateTime time.Time      `json:"updateTime"`
	CreateBy   *int64         `json:"createBy"`
	UpdateBy   *int64         `json:"updateBy"`
	Remark     string         `json:"remark"`
	DeletedAt  *time.Time     `json:"deletedAt"`
}

func (SysEmployee) TableName() string {
	return "sys_employee"
}

// SysLoginLog 登录日志表
type SysLoginLog struct {
	ID         int64     `json:"id"`
	EmployeeID *int64    `json:"employeeId"`
	Phone      string    `json:"phone"`
	IP         string    `json:"ip"`
	Status     int8      `json:"status"`
	Msg        string    `json:"msg"`
	LoginTime  time.Time `json:"loginTime"`
}

func (SysLoginLog) TableName() string {
	return "sys_login_log"
}
