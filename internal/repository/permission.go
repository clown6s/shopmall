package repository

import (
	"context"
	"database/sql"
	"shopmall/internal/model"
)

// PermissionRepo 权限数据访问层
type PermissionRepo struct {
	db *sql.DB
}

// NewPermissionRepo 创建 PermissionRepo 实例
func NewPermissionRepo(db *sql.DB) *PermissionRepo {
	return &PermissionRepo{db: db}
}

// Create 创建权限
func (r *PermissionRepo) Create(ctx context.Context, perm *model.SysPermission) error {
	query := `INSERT INTO sys_permission (permission_name, permission_key, type, parent_id, path, method, sort_order, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		perm.PermissionName, perm.PermissionKey, perm.Type, perm.ParentID,
		perm.Path, perm.Method, perm.SortOrder, perm.Status)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	perm.ID = id
	return nil
}

// GetByID 根据ID获取权限
func (r *PermissionRepo) GetByID(ctx context.Context, id int64) (*model.SysPermission, error) {
	query := `SELECT id, permission_name, permission_key, type, parent_id, path, method, sort_order, status, created_at, updated_at
		FROM sys_permission WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	var perm model.SysPermission
	err := row.Scan(&perm.ID, &perm.PermissionName, &perm.PermissionKey, &perm.Type,
		&perm.ParentID, &perm.Path, &perm.Method, &perm.SortOrder,
		&perm.Status, &perm.CreatedAt, &perm.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

// ListAll 获取所有权限（按排序升序）
func (r *PermissionRepo) ListAll(ctx context.Context) ([]*model.SysPermission, error) {
	query := `SELECT id, permission_name, permission_key, type, parent_id, path, method, sort_order, status, created_at, updated_at
		FROM sys_permission ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.SysPermission
	for rows.Next() {
		var perm model.SysPermission
		err := rows.Scan(&perm.ID, &perm.PermissionName, &perm.PermissionKey, &perm.Type,
			&perm.ParentID, &perm.Path, &perm.Method, &perm.SortOrder,
			&perm.Status, &perm.CreatedAt, &perm.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &perm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// GetMenuPermissionsByRoleID 根据角色ID获取菜单权限
// 不仅返回直接分配的菜单(type=1)，还返回被分配的子权限（按钮/API）的父级菜单路径
func (r *PermissionRepo) GetMenuPermissionsByRoleID(ctx context.Context, roleID int64) ([]*model.SysPermission, error) {
	query := `SELECT DISTINCT p.id, p.permission_name, p.permission_key, p.type, p.parent_id, p.path, p.method, p.sort_order, p.status, p.created_at, p.updated_at
		FROM sys_permission p
		WHERE p.status = 1 AND p.type = 1 AND p.path != '' AND (
			-- 直接分配的菜单权限
			p.id IN (SELECT permission_id FROM sys_role_permission WHERE role_id = ?)
			-- 被分配的子权限（按钮/API）的父菜单
			OR p.id IN (
				SELECT DISTINCT child.parent_id
				FROM sys_role_permission rp
				INNER JOIN sys_permission child ON rp.permission_id = child.id
				WHERE rp.role_id = ? AND child.parent_id != 0
			)
		)
		ORDER BY p.sort_order ASC, p.id ASC`
	rows, err := r.db.QueryContext(ctx, query, roleID, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.SysPermission
	for rows.Next() {
		var perm model.SysPermission
		err := rows.Scan(&perm.ID, &perm.PermissionName, &perm.PermissionKey, &perm.Type,
			&perm.ParentID, &perm.Path, &perm.Method, &perm.SortOrder,
			&perm.Status, &perm.CreatedAt, &perm.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &perm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// GetButtonPermissionKeysByRoleID 根据角色ID获取操作权限key列表（type=2按钮 + type=3 API）
func (r *PermissionRepo) GetButtonPermissionKeysByRoleID(ctx context.Context, roleID int64) ([]string, error) {
	query := `SELECT p.permission_key
		FROM sys_permission p
		INNER JOIN sys_role_permission rp ON p.id = rp.permission_id
		WHERE rp.role_id = ? AND p.type IN (2, 3) AND p.status = 1`
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

// Update 更新权限
func (r *PermissionRepo) Update(ctx context.Context, perm *model.SysPermission) error {
	query := `UPDATE sys_permission SET permission_name = ?, permission_key = ?, type = ?,
		parent_id = ?, path = ?, method = ?, sort_order = ?, status = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query,
		perm.PermissionName, perm.PermissionKey, perm.Type, perm.ParentID,
		perm.Path, perm.Method, perm.SortOrder, perm.Status, perm.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete 删除权限
func (r *PermissionRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sys_permission WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteWithRoleRefs 事务性删除权限及其角色关联引用
func (r *PermissionRepo) DeleteWithRoleRefs(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM sys_role_permission WHERE permission_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sys_permission WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
