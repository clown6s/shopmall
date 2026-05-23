package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"shopmall/internal/model"
	"shopmall/pkg"

	"go.uber.org/zap"
)

// EmployeeRepo 员工数据访问层
type EmployeeRepo struct {
	db *sql.DB
}

// NewEmployeeRepo 创建 EmployeeRepo 实例
func NewEmployeeRepo(db *sql.DB) *EmployeeRepo {
	return &EmployeeRepo{db: db}
}

// GetByPhone 按手机号查询员工（含密码）
func (r *EmployeeRepo) GetByPhone(ctx context.Context, phone string) (*model.SysEmployee, error) {
	const query = `SELECT employee_id, password, real_name, avatar, gender, age, phone, email,
		department, position, role, work_status, can_login, create_time, update_time,
		create_by, update_by, remark, deleted_at
		FROM sys_employee WHERE phone = ? AND deleted_at IS NULL LIMIT 1`

	return r.scanOne(ctx, query, phone)
}

// GetByID 按 ID 查询员工（含密码）
func (r *EmployeeRepo) GetByID(ctx context.Context, id int64) (*model.SysEmployee, error) {
	const query = `SELECT employee_id, password, real_name, avatar, gender, age, phone, email,
		department, position, role, work_status, can_login, create_time, update_time,
		create_by, update_by, remark, deleted_at
		FROM sys_employee WHERE employee_id = ? AND deleted_at IS NULL LIMIT 1`

	return r.scanOne(ctx, query, id)
}

// GetByIDWithoutPassword 按 ID 查询员工（不含密码）
func (r *EmployeeRepo) GetByIDWithoutPassword(ctx context.Context, id int64) (*model.SysEmployee, error) {
	const query = `SELECT employee_id, real_name, avatar, gender, age, phone, email,
		department, position, role, work_status, can_login, create_time, update_time,
		create_by, update_by, remark, deleted_at
		FROM sys_employee WHERE employee_id = ? AND deleted_at IS NULL LIMIT 1`

	return r.scanOneWithoutPassword(ctx, query, id)
}

// InsertEmployee 插入新员工记录
func (r *EmployeeRepo) InsertEmployee(ctx context.Context, emp *model.SysEmployee) error {
	const query = `INSERT INTO sys_employee
		(password, real_name, avatar, gender, age, phone, email, department, position, role,
		 work_status, can_login, create_by, update_by, remark)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query,
		emp.Password, emp.RealName, emp.Avatar, emp.Gender, emp.Age,
		emp.Phone, emp.Email, emp.Department, emp.Position, emp.Role,
		emp.WorkStatus, emp.CanLogin, emp.CreateBy, emp.UpdateBy, emp.Remark,
	)
	if err != nil {
		return fmt.Errorf("insert employee failed: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id failed: %w", err)
	}
	emp.EmployeeID = id
	return nil
}

// UpdateEmployee 更新员工信息
func (r *EmployeeRepo) UpdateEmployee(ctx context.Context, emp *model.SysEmployee) error {
	const query = `UPDATE sys_employee SET
		real_name = ?, avatar = ?, gender = ?, age = ?, phone = ?, email = ?,
		department = ?, position = ?, role = ?, work_status = ?, can_login = ?,
		update_by = ?, remark = ?
		WHERE employee_id = ? AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query,
		emp.RealName, emp.Avatar, emp.Gender, emp.Age,
		emp.Phone, emp.Email, emp.Department, emp.Position, emp.Role,
		emp.WorkStatus, emp.CanLogin, emp.UpdateBy, emp.Remark, emp.EmployeeID,
	)
	if err != nil {
		return fmt.Errorf("update employee failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected check failed: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdatePassword 更新员工密码
func (r *EmployeeRepo) UpdatePassword(ctx context.Context, empID int64, hash string) error {
	const query = `UPDATE sys_employee SET password = ?
		WHERE employee_id = ? AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, hash, empID)
	if err != nil {
		return fmt.Errorf("update password failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected check failed: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListEmployee 分页查询员工列表（模糊匹配姓名和手机号）
func (r *EmployeeRepo) ListEmployee(ctx context.Context, realName, phone string, page, pageSize int) ([]*model.SysEmployee, int64, error) {
	// 构建 WHERE 条件
	var conditions []string
	var args []interface{}

	if realName != "" {
		conditions = append(conditions, "real_name LIKE ?")
		args = append(args, "%"+realName+"%")
	}
	if phone != "" {
		conditions = append(conditions, "phone LIKE ?")
		args = append(args, "%"+phone+"%")
	}

	where := "WHERE deleted_at IS NULL"
	if len(conditions) > 0 {
		where += " AND " + strings.Join(conditions, " AND ")
	}

	// 查询总数
	var total int64
	countQuery := "SELECT COUNT(*) FROM sys_employee " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count employees failed: %w", err)
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询列表
	listQuery := fmt.Sprintf(`SELECT employee_id, real_name, avatar, gender, age, phone, email,
		department, position, role, work_status, can_login, create_time, update_time,
		create_by, update_by, remark, deleted_at
		FROM sys_employee %s ORDER BY employee_id DESC LIMIT ? OFFSET ?`, where)

	// 将 LIMIT/OFFSET 参数追加到 args
	listArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query employees failed: %w", err)
	}
	defer rows.Close()

	var list []*model.SysEmployee
	for rows.Next() {
		emp := &model.SysEmployee{}
		if err := r.scanRows(rows, emp, false); err != nil {
			return nil, 0, fmt.Errorf("scan employee row failed: %w", err)
		}
		list = append(list, emp)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate employee rows failed: %w", err)
	}

	return list, total, nil
}

// InsertLoginLog 插入登录日志
func (r *EmployeeRepo) InsertLoginLog(ctx context.Context, log *model.SysLoginLog) error {
	const query = `INSERT INTO sys_login_log (employee_id, phone, ip, status, msg)
		VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		log.EmployeeID, log.Phone, log.IP, log.Status, log.Msg,
	)
	if err != nil {
		return fmt.Errorf("insert login log failed: %w", err)
	}
	return nil
}

// ===================== 内部扫描方法 =====================

// scanOne 扫描一行记录（含密码）
func (r *EmployeeRepo) scanOne(ctx context.Context, query string, args ...interface{}) (*model.SysEmployee, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	emp := &model.SysEmployee{}
	if err := r.scanRow(row, emp, true); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		pkg.Logger.Error("[EmployeeRepo] scanOne failed", zap.Error(err), zap.String("query", query))
		return nil, err
	}
	return emp, nil
}

// scanOneWithoutPassword 扫描一行记录（不含密码）
func (r *EmployeeRepo) scanOneWithoutPassword(ctx context.Context, query string, args ...interface{}) (*model.SysEmployee, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	emp := &model.SysEmployee{}
	if err := r.scanRow(row, emp, false); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		pkg.Logger.Error("[EmployeeRepo] scanOneWithoutPassword failed", zap.Error(err))
		return nil, err
	}
	return emp, nil
}

// scanRow 从 *sql.Row 扫描员工记录
func (r *EmployeeRepo) scanRow(row *sql.Row, emp *model.SysEmployee, withPassword bool) error {
	if withPassword {
		return row.Scan(
			&emp.EmployeeID, &emp.Password, &emp.RealName, &emp.Avatar,
			&emp.Gender, &emp.Age, &emp.Phone, &emp.Email,
			&emp.Department, &emp.Position, &emp.Role,
			&emp.WorkStatus, &emp.CanLogin, &emp.CreateTime, &emp.UpdateTime,
			&emp.CreateBy, &emp.UpdateBy, &emp.Remark, &emp.DeletedAt,
		)
	}
	return row.Scan(
		&emp.EmployeeID, &emp.RealName, &emp.Avatar,
		&emp.Gender, &emp.Age, &emp.Phone, &emp.Email,
		&emp.Department, &emp.Position, &emp.Role,
		&emp.WorkStatus, &emp.CanLogin, &emp.CreateTime, &emp.UpdateTime,
		&emp.CreateBy, &emp.UpdateBy, &emp.Remark, &emp.DeletedAt,
	)
}

// scanRows 从 *sql.Rows 扫描员工记录
func (r *EmployeeRepo) scanRows(rows *sql.Rows, emp *model.SysEmployee, withPassword bool) error {
	if withPassword {
		return rows.Scan(
			&emp.EmployeeID, &emp.Password, &emp.RealName, &emp.Avatar,
			&emp.Gender, &emp.Age, &emp.Phone, &emp.Email,
			&emp.Department, &emp.Position, &emp.Role,
			&emp.WorkStatus, &emp.CanLogin, &emp.CreateTime, &emp.UpdateTime,
			&emp.CreateBy, &emp.UpdateBy, &emp.Remark, &emp.DeletedAt,
		)
	}
	return rows.Scan(
		&emp.EmployeeID, &emp.RealName, &emp.Avatar,
		&emp.Gender, &emp.Age, &emp.Phone, &emp.Email,
		&emp.Department, &emp.Position, &emp.Role,
		&emp.WorkStatus, &emp.CanLogin, &emp.CreateTime, &emp.UpdateTime,
		&emp.CreateBy, &emp.UpdateBy, &emp.Remark, &emp.DeletedAt,
	)
}
