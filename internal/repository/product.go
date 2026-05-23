package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"shopmall/common"
	"shopmall/internal/model"
	"shopmall/pkg"

	"go.uber.org/zap"
)

// ==================== Product (Category + Brand) ====================

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

// GetAllCategories 查询所有启用的分类
func (r *ProductRepo) GetAllCategories(ctx context.Context) ([]model.Category, error) {
	query := `SELECT cat_id, name, parent_cid, cat_level, show_status, sort, icon
			  FROM category WHERE deleted_at IS NULL AND show_status = 1 ORDER BY sort ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		pkg.Logger.Error("查询商品分类失败：", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	cats := make([]model.Category, 0)
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.CatID, &c.Name, &c.ParentCid, &c.CatLevel, &c.ShowStatus, &c.Sort, &c.Icon); err != nil {
			pkg.Logger.Error("扫描商品分类失败：", zap.Error(err))
			return nil, err
		}
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		pkg.Logger.Error("遍历商品分类失败：", zap.Error(err))
		return nil, err
	}
	return cats, nil
}

// GetCategoryByID 根据ID查询分类
func (r *ProductRepo) GetCategoryByID(ctx context.Context, catID int64) (*model.Category, error) {
	query := `SELECT cat_id, name, parent_cid, cat_level, show_status, sort, icon
			  FROM category WHERE cat_id = ? AND deleted_at IS NULL`
	var cat model.Category
	if err := r.db.QueryRowContext(ctx, query, catID).
		Scan(&cat.CatID, &cat.Name, &cat.ParentCid, &cat.CatLevel, &cat.ShowStatus, &cat.Sort, &cat.Icon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		pkg.Logger.Error("查询商品分类失败", zap.Int64("cat_id", catID), zap.Error(err))
		return nil, err
	}
	return &cat, nil
}

// AddCategory 添加分类
func (r *ProductRepo) AddCategory(ctx context.Context, cat *model.Category) (int64, error) {
	query := `INSERT INTO category (name, parent_cid, cat_level, show_status, sort, icon) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, cat.Name, cat.ParentCid, cat.CatLevel, cat.ShowStatus, cat.Sort, cat.Icon)
	if err != nil {
		pkg.Logger.Error("添加分类失败：", zap.Error(err))
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		pkg.Logger.Error("获取新增分类ID失败：", zap.Error(err))
		return 0, err
	}
	return id, nil
}

// EditCategory 更新分类（只更新 name, sort, show_status）
func (r *ProductRepo) EditCategory(ctx context.Context, catID int64, name string, sort int, showStatus int8) error {
	query := `UPDATE category SET name = ?, sort = ?, show_status = ? WHERE cat_id = ? AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, name, sort, showStatus, catID)
	if err != nil {
		pkg.Logger.Error("更新分类失败：", zap.Int64("cat_id", catID), zap.Error(err))
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

// DeleteCategory 逻辑删除分类（包括所有子分类和关联品牌）
func (r *ProductRepo) DeleteCategory(ctx context.Context, catID int64) (int64, int64, error) {
	// 开启事务（BFS 读取在事务内，避免 TOCTOU 竞态）
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	committed := false
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		} else if !committed {
			tx.Rollback()
		}
	}()

	type node struct {
		ID       int64
		ParentID int64
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT cat_id, parent_cid FROM category WHERE deleted_at IS NULL`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	childrenMap := make(map[int64][]int64)
	for rows.Next() {
		var n node
		if err := rows.Scan(&n.ID, &n.ParentID); err != nil {
			return 0, 0, err
		}
		childrenMap[n.ParentID] = append(childrenMap[n.ParentID], n.ID)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	// BFS 找所有子节点（带环检测和深度限制）
	var ids []int64
	visited := make(map[int64]bool)
	queue := []int64{catID}
	maxDepth := 10
	for len(queue) > 0 && maxDepth > 0 {
		maxDepth--
		var next []int64
		for _, id := range queue {
			if visited[id] {
				continue
			}
			visited[id] = true
			ids = append(ids, id)
			for _, child := range childrenMap[id] {
				if !visited[child] {
					next = append(next, child)
				}
			}
		}
		queue = next
	}
	if len(ids) == 0 {
		return 0, 0, ErrNotFound
	}

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	placeholders := buildPlaceholders(len(ids))

	// 删除分类
	catSQL := "UPDATE category SET deleted_at = NOW() WHERE cat_id IN (" + placeholders + ")"
	res1, err := tx.ExecContext(ctx, catSQL, args...)
	if err != nil {
		return 0, 0, err
	}
	catRows, err := res1.RowsAffected()
	if err != nil {
		return 0, 0, fmt.Errorf("获取影响行数失败: %w", err)
	}

	// 删除品牌（物理删除，brand 表无 deleted_at 列）
	brandSQL := "DELETE FROM brand WHERE cat_id IN (" + placeholders + ")"
	res2, err := tx.ExecContext(ctx, brandSQL, args...)
	if err != nil {
		return 0, 0, err
	}
	brandRows, err := res2.RowsAffected()
	if err != nil {
		return 0, 0, fmt.Errorf("获取影响行数失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	committed = true

	return catRows, brandRows, nil
}

// GetBrandsByCatID 根据分类ID获取品牌列表
func (r *ProductRepo) GetBrandsByCatID(ctx context.Context, catID int64) ([]model.Brand, error) {
	query := `SELECT brand_id, name, logo, description, show_status, first_letter, sort, cat_id
			  FROM brand WHERE cat_id = ? ORDER BY sort ASC`
	rows, err := r.db.QueryContext(ctx, query, catID)
	if err != nil {
		pkg.Logger.Error("查询品牌列表失败", zap.Int64("cat_id", catID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var brands []model.Brand
	for rows.Next() {
		var b model.Brand
		if err := rows.Scan(&b.BrandID, &b.Name, &b.Logo, &b.Description, &b.ShowStatus, &b.FirstLetter, &b.Sort, &b.CatId); err != nil {
			pkg.Logger.Error("扫描品牌失败", zap.Int64("cat_id", catID), zap.Error(err))
			return nil, err
		}
		brands = append(brands, b)
	}
	if err := rows.Err(); err != nil {
		pkg.Logger.Error("遍历品牌失败", zap.Int64("cat_id", catID), zap.Error(err))
		return nil, err
	}
	return brands, nil
}

// GetBrandByID 根据ID查询品牌
func (r *ProductRepo) GetBrandByID(ctx context.Context, brandID int64) (*model.Brand, error) {
	query := `SELECT brand_id, name, logo, description, show_status, first_letter, sort, cat_id
			  FROM brand WHERE brand_id = ?`
	var brand model.Brand
	if err := r.db.QueryRowContext(ctx, query, brandID).Scan(
		&brand.BrandID, &brand.Name, &brand.Logo, &brand.Description,
		&brand.ShowStatus, &brand.FirstLetter, &brand.Sort, &brand.CatId,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		pkg.Logger.Error("查询品牌失败：", zap.Int64("brand_id", brandID), zap.Error(err))
		return nil, err
	}
	return &brand, nil
}

// AddBrand 添加品牌
func (r *ProductRepo) AddBrand(ctx context.Context, brand *model.Brand) (int64, error) {
	brandID, err := common.NextBrandID()
	if err != nil {
		return 0, fmt.Errorf("生成品牌ID失败: %w", err)
	}
	query := `INSERT INTO brand (brand_id, cat_id, name, logo, description, show_status, first_letter, sort) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		brandID, brand.CatId, brand.Name, brand.Logo, brand.Description, brand.ShowStatus, brand.FirstLetter, brand.Sort)
	if err != nil {
		pkg.Logger.Error("添加品牌失败：", zap.Error(err))
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	if rows == 0 {
		return 0, errors.New("insert failed")
	}
	return brandID, nil
}

// UpdateBrand 更新品牌（动态SQL，只更新有值的字段）
func (r *ProductRepo) UpdateBrand(ctx context.Context, brandID int64, brand *model.Brand) error {
	var setParts []string
	var args []interface{}

	if brand.Name != "" {
		setParts = append(setParts, "name = ?")
		args = append(args, brand.Name)
	}
	if brand.Logo != "" {
		setParts = append(setParts, "logo = ?")
		args = append(args, brand.Logo)
	}
	if brand.Description != "" {
		setParts = append(setParts, "description = ?")
		args = append(args, brand.Description)
	}
	if brand.ShowStatus != nil {
		setParts = append(setParts, "show_status = ?")
		args = append(args, *brand.ShowStatus)
	}
	if brand.FirstLetter != "" {
		setParts = append(setParts, "first_letter = ?")
		args = append(args, brand.FirstLetter)
	}
	// TODO: Brand.Sort uses int, can't distinguish "unset" from "set to 0". Consider using *int.
	if brand.Sort > 0 {
		setParts = append(setParts, "sort = ?")
		args = append(args, brand.Sort)
	}
	if brand.CatId > 0 {
		setParts = append(setParts, "cat_id = ?")
		args = append(args, brand.CatId)
	}

	if len(setParts) == 0 {
		return nil
	}

	query := "UPDATE brand SET " + strings.Join(setParts, ", ") + " WHERE brand_id = ?"
	args = append(args, brandID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		pkg.Logger.Error("更新品牌失败：", zap.Error(err))
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

// DeleteBrand 删除品牌（物理删除，brand 表无 deleted_at 列）
func (r *ProductRepo) DeleteBrand(ctx context.Context, brandID int64) error {
	query := `DELETE FROM brand WHERE brand_id = ?`
	result, err := r.db.ExecContext(ctx, query, brandID)
	if err != nil {
		pkg.Logger.Error("删除品牌失败：", zap.Error(err))
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

// CategoryExists 检查分类是否存在
func (r *ProductRepo) CategoryExists(ctx context.Context, catID int64) (bool, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM category WHERE cat_id = ? AND deleted_at IS NULL", catID).Scan(&count)
	return count > 0, err
}
