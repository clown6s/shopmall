package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"shopmall/internal/model"
	"shopmall/pkg"

	"go.uber.org/zap"
)

// ErrNotFound 记录不存在
var ErrNotFound = errors.New("record not found")

// escapeLike 转义 LIKE 查询中的特殊字符
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// buildPlaceholders 安全生成 SQL 占位符字符串
func buildPlaceholders(n int) string {
	if n <= 0 {
		return "'__NEVER_MATCH__'"
	}
	result := make([]string, n)
	for i := range result {
		result[i] = "?"
	}
	return strings.Join(result, ",")
}

// ==================== Department ====================

type DepartmentRepo struct {
	db *sql.DB
}

func NewDepartmentRepo(db *sql.DB) *DepartmentRepo {
	return &DepartmentRepo{db: db}
}

// GetAll 查询所有部门
func (r *DepartmentRepo) GetAll(ctx context.Context) ([]model.SysDepartment, error) {
	query := `SELECT dept_id, dept_name, dept_sort, status, create_time, update_time, remark
			  FROM sys_department ORDER BY dept_sort ASC, dept_id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		pkg.Logger.Error("查询部门列表失败", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var list []model.SysDepartment
	for rows.Next() {
		var d model.SysDepartment
		if err := rows.Scan(&d.DeptID, &d.DeptName, &d.DeptSort, &d.Status, &d.CreateTime, &d.UpdateTime, &d.Remark); err != nil {
			pkg.Logger.Error("扫描部门数据失败", zap.Error(err))
			return nil, err
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		pkg.Logger.Error("遍历部门数据失败", zap.Error(err))
		return nil, err
	}
	return list, nil
}

// GetAllWithPage 分页查询部门
func (r *DepartmentRepo) GetAllWithPage(ctx context.Context, page, pageSize int, deptName string) ([]model.SysDepartment, int, error) {
	var total int
	countArgs := make([]interface{}, 0)
	countQuery := `SELECT COUNT(*) FROM sys_department WHERE 1=1`
	if deptName != "" {
		countQuery += ` AND dept_name LIKE ? ESCAPE '\\'`
		countArgs = append(countArgs, "%"+escapeLike(deptName)+"%")
	}
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		pkg.Logger.Error("统计部门数量失败", zap.Error(err))
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	pageArgs := make([]interface{}, 0)
	query := `SELECT dept_id, dept_name, dept_sort, status, create_time, update_time, remark
			  FROM sys_department WHERE 1=1`
	if deptName != "" {
		query += ` AND dept_name LIKE ? ESCAPE '\\'`
		pageArgs = append(pageArgs, "%"+escapeLike(deptName)+"%")
	}
	query += ` ORDER BY dept_sort ASC, dept_id ASC LIMIT ? OFFSET ?`
	pageArgs = append(pageArgs, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, pageArgs...)
	if err != nil {
		pkg.Logger.Error("分页查询部门失败", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var list []model.SysDepartment
	for rows.Next() {
		var d model.SysDepartment
		if err := rows.Scan(&d.DeptID, &d.DeptName, &d.DeptSort, &d.Status, &d.CreateTime, &d.UpdateTime, &d.Remark); err != nil {
			pkg.Logger.Error("扫描部门数据失败", zap.Error(err))
			return nil, 0, err
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		pkg.Logger.Error("遍历部门数据失败", zap.Error(err))
		return nil, 0, err
	}
	return list, total, nil
}

// GetByID 根据ID查询部门
func (r *DepartmentRepo) GetByID(ctx context.Context, deptID int64) (*model.SysDepartment, error) {
	query := `SELECT dept_id, dept_name, dept_sort, status, create_time, update_time, remark
			  FROM sys_department WHERE dept_id = ?`
	var d model.SysDepartment
	if err := r.db.QueryRowContext(ctx, query, deptID).Scan(&d.DeptID, &d.DeptName, &d.DeptSort, &d.Status, &d.CreateTime, &d.UpdateTime, &d.Remark); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		pkg.Logger.Error("查询部门失败", zap.Int64("deptId", deptID), zap.Error(err))
		return nil, err
	}
	return &d, nil
}

// Create 新增部门
func (r *DepartmentRepo) Create(ctx context.Context, dept *model.SysDepartment) (int64, error) {
	query := `INSERT INTO sys_department (dept_name, dept_sort, status, remark) VALUES (?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, dept.DeptName, dept.DeptSort, dept.Status, dept.Remark)
	if err != nil {
		pkg.Logger.Error("新增部门失败", zap.Error(err))
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		pkg.Logger.Error("获取新增部门ID失败", zap.Error(err))
		return 0, err
	}
	return id, nil
}

// Update 更新部门
func (r *DepartmentRepo) Update(ctx context.Context, deptID int64, dept *model.SysDepartment) error {
	query := `UPDATE sys_department SET dept_name = ?, dept_sort = ?, status = ?, remark = ? WHERE dept_id = ?`
	result, err := r.db.ExecContext(ctx, query, dept.DeptName, dept.DeptSort, dept.Status, dept.Remark, deptID)
	if err != nil {
		pkg.Logger.Error("更新部门失败", zap.Int64("deptId", deptID), zap.Error(err))
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete 删除部门（事务防竞态）
func (r *DepartmentRepo) Delete(ctx context.Context, deptID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var count int
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sys_employee e INNER JOIN sys_department d ON e.department = d.dept_name WHERE d.dept_id = ? AND e.deleted_at IS NULL FOR UPDATE`,
		deptID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该部门下还有 %d 名员工，无法删除", count)
	}

	result, err := tx.ExecContext(ctx, `UPDATE sys_department SET deleted_at = NOW() WHERE dept_id = ? AND deleted_at IS NULL`, deptID)
	if err != nil {
		pkg.Logger.Error("删除部门失败", zap.Int64("deptId", deptID), zap.Error(err))
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}
