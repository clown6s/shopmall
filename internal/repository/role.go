package repository

import (
	"context"
	"database/sql"
	"shopmall/internal/model"
	"strings"
)

// RoleRepo 角色数据访问层
type RoleRepo struct {
	db *sql.DB
}

// NewRoleRepo 创建 RoleRepo 实例
func NewRoleRepo(db *sql.DB) *RoleRepo {
	return &RoleRepo{db: db}
}

// Create 创建角色
func (r *RoleRepo) Create(ctx context.Context, role *model.SysRole) error {
	query := `INSERT INTO sys_role (role_name, role_key, description, status)
		VALUES (?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		role.RoleName, role.RoleKey, role.Description, role.Status)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	role.ID = id
	return nil
}

// GetByID 根据ID获取角色
func (r *RoleRepo) GetByID(ctx context.Context, id int64) (*model.SysRole, error) {
	query := `SELECT id, role_name, role_key, description, status, created_at, updated_at
		FROM sys_role WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	var role model.SysRole
	err := row.Scan(&role.ID, &role.RoleName, &role.RoleKey, &role.Description,
		&role.Status, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// ListAll 获取所有角色
func (r *RoleRepo) ListAll(ctx context.Context) ([]*model.SysRole, error) {
	query := `SELECT id, role_name, role_key, description, status, created_at, updated_at
		FROM sys_role ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.SysRole
	for rows.Next() {
		var role model.SysRole
		err := rows.Scan(&role.ID, &role.RoleName, &role.RoleKey, &role.Description,
			&role.Status, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// Update 更新角色
func (r *RoleRepo) Update(ctx context.Context, role *model.SysRole) error {
	query := `UPDATE sys_role SET role_name = ?, role_key = ?, description = ?, status = ?
		WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query,
		role.RoleName, role.RoleKey, role.Description, role.Status, role.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete 删除角色
func (r *RoleRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM sys_role WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetByRoleKey 根据角色标识获取角色
func (r *RoleRepo) GetByRoleKey(ctx context.Context, roleKey string) (*model.SysRole, error) {
	query := `SELECT id, role_name, role_key, description, status, created_at, updated_at
		FROM sys_role WHERE role_key = ? AND status = 1`
	row := r.db.QueryRowContext(ctx, query, roleKey)
	var role model.SysRole
	err := row.Scan(&role.ID, &role.RoleName, &role.RoleKey, &role.Description,
		&role.Status, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByRoleName 根据角色名称获取角色
func (r *RoleRepo) GetByRoleName(ctx context.Context, roleName string) (*model.SysRole, error) {
	query := `SELECT id, role_name, role_key, description, status, created_at, updated_at
		FROM sys_role WHERE role_name = ?`
	row := r.db.QueryRowContext(ctx, query, roleName)
	var role model.SysRole
	err := row.Scan(&role.ID, &role.RoleName, &role.RoleKey, &role.Description,
		&role.Status, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetPermissionIDs 获取角色的权限ID列表
func (r *RoleRepo) GetPermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	query := `SELECT permission_id FROM sys_role_permission WHERE role_id = ?`
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// SetPermissions 设置角色权限（事务：先删后插）
func (r *RoleRepo) SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除旧关联
	if _, err := tx.ExecContext(ctx, `DELETE FROM sys_role_permission WHERE role_id = ?`, roleID); err != nil {
		return err
	}

	// 插入新关联
	if len(permissionIDs) > 0 {
		placeholders := make([]string, len(permissionIDs))
		args := make([]interface{}, 0, len(permissionIDs)*2)
		for i, permID := range permissionIDs {
			placeholders[i] = "(?, ?)"
			args = append(args, roleID, permID)
		}
		query := "INSERT INTO sys_role_permission (role_id, permission_id) VALUES " + strings.Join(placeholders, ", ")
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteWithPermissions 事务性删除角色及其权限关联
func (r *RoleRepo) DeleteWithPermissions(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM sys_role_permission WHERE role_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sys_role WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
