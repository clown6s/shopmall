package service

import (
	"context"

	"shopmall/internal/model"
	"shopmall/internal/repository"
)

// DepartmentService 部门业务逻辑
type DepartmentService struct {
	deptRepo *repository.DepartmentRepo
}

func NewDepartmentSvc(repo *repository.DepartmentRepo) *DepartmentService {
	return &DepartmentService{deptRepo: repo}
}

// GetAll 获取所有部门
func (s *DepartmentService) GetAll(ctx context.Context) ([]model.SysDepartment, error) {
	return s.deptRepo.GetAll(ctx)
}

// GetAllWithPage 分页获取部门
func (s *DepartmentService) GetAllWithPage(ctx context.Context, page, pageSize int, deptName string) ([]model.SysDepartment, int, error) {
	return s.deptRepo.GetAllWithPage(ctx, page, pageSize, deptName)
}

// GetByID 根据ID获取部门
func (s *DepartmentService) GetByID(ctx context.Context, deptID int64) (*model.SysDepartment, error) {
	return s.deptRepo.GetByID(ctx, deptID)
}

// Create 新增部门
func (s *DepartmentService) Create(ctx context.Context, dept *model.SysDepartment) (int64, error) {
	if dept.DeptName == "" {
		return 0, ErrInvalidParam
	}
	if dept.Status == 0 {
		dept.Status = 1
	}
	return s.deptRepo.Create(ctx, dept)
}

// Update 更新部门
func (s *DepartmentService) Update(ctx context.Context, deptID int64, dept *model.SysDepartment) error {
	if dept.DeptName == "" {
		return ErrInvalidParam
	}
	return s.deptRepo.Update(ctx, deptID, dept)
}

// Delete 删除部门
func (s *DepartmentService) Delete(ctx context.Context, deptID int64) error {
	return s.deptRepo.Delete(ctx, deptID)
}
