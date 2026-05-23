package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"shopmall/common"
	"shopmall/common/middleware"
	commonutil "shopmall/common/util"
	"shopmall/internal/config"
	"shopmall/internal/dto"
	"shopmall/internal/model"
	"shopmall/internal/service"
	"shopmall/pkg"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	refreshTokenBlacklistPrefix = "rt:blacklist:"
	accessTokenBlacklistPrefix  = "at:blacklist:"
	refreshLockPrefix           = "rt:lock:"
	refreshFamilyPrefix         = "rt:family:"
)

// ErrWrongOldPassword 原密码错误
var ErrWrongOldPassword = errors.New("原密码错误")

// EmployeeHandler 员工控制器
type EmployeeHandler struct {
	EmployeeService  *service.EmployeeService
	RoleService      *service.RoleService
	LoginRateLimiter *middleware.LoginRateLimiter
	Redis            *redis.Client
}

// NewEmployeeHandler 创建 EmployeeHandler 实例
func NewEmployeeHandler(svc *service.EmployeeService, roleSvc *service.RoleService, rateLimiter *middleware.LoginRateLimiter, redisClient *redis.Client) *EmployeeHandler {
	return &EmployeeHandler{
		EmployeeService:  svc,
		RoleService:      roleSvc,
		LoginRateLimiter: rateLimiter,
		Redis:            redisClient,
	}
}

// ===================== 登录/登出 =====================

// Login 员工登录
// POST /api/employee/login
func (h *EmployeeHandler) Login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if req.Phone == "" || req.Password == "" {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "手机号和密码不能为空"))
		return
	}

	if !service.IsValidPhone(req.Phone) {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "手机号格式不正确"))
		return
	}

	if len(req.Password) < model.PasswordMinLen || len(req.Password) > model.PasswordMaxLen {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, fmt.Sprintf("密码长度必须在%d-%d位之间", model.PasswordMinLen, model.PasswordMaxLen)))
		return
	}

	// 登录速率限制检查
	if h.LoginRateLimiter != nil {
		allowed, _, waitSeconds := h.LoginRateLimiter.AllowCheck(ctx, req.Phone)
		if !allowed {
			waitTime := time.Duration(waitSeconds) * time.Second
			pkg.Logger.Warn("[Login] 账号已被锁定", zap.String("phone", service.MaskPhone(req.Phone)), zap.Duration("wait", waitTime))
			common.WriteJSON(c, http.StatusTooManyRequests, common.Fail(429, "登录尝试次数过多，请 "+waitTime.Round(time.Second).String()+" 后再试"))
			return
		}
	}

	loginIP := c.ClientIP()

	resp, token, refreshToken, err := h.EmployeeService.Login(ctx, &req, loginIP)
	if err != nil {
		if h.LoginRateLimiter != nil {
			_, _, _ = h.LoginRateLimiter.RecordFail(ctx, req.Phone)
		}

		switch err {
		case service.ErrInvalidCredentials, service.ErrEmployeeDisabled, service.ErrCannotLogin:
			common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "手机号或密码错误"))
		default:
			pkg.Logger.Error("Login failed", zap.Error(err))
			common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "登录失败，请重试"))
		}
		return
	}

	// 登录成功，清除失败记录
	if h.LoginRateLimiter != nil {
		_ = h.LoginRateLimiter.ClearFail(ctx, req.Phone)
	}

	// 设置 JWT httpOnly Cookie
	setAuthCookie(c, token)

	// 设置 Refresh Token Cookie
	if refreshToken != "" {
		setRefreshCookie(c, refreshToken)
	}

	// 设置 CSRF Cookie
	middleware.SetCSRFCookie(nil, c, token)

	common.WriteJSON(c, http.StatusOK, common.OkData(resp))
}

// Logout 员工登出
// POST /api/auth/logout
func (h *EmployeeHandler) Logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	jwtToken, _ := c.Cookie("auth_token")
	if jwtToken != "" {
		middleware.InvalidateCSRF(jwtToken)

		// 将 access token 加入 Redis 黑名单
		if h.Redis != nil {
			ttl := getAccessTokenTTL(jwtToken)
			blacklistKey := accessTokenBlacklistPrefix + commonutil.HashSHA256(jwtToken)
			if err := h.Redis.Set(ctx, blacklistKey, "1", ttl).Err(); err != nil {
				pkg.Logger.Warn("[Logout] Redis access token blacklist set failed", zap.Error(err))
			}
		}
	}

	// 将 refresh token 加入 Redis 黑名单
	refreshTokenStr, _ := c.Cookie("refresh_token")
	if refreshTokenStr != "" && h.Redis != nil {
		ttl := calcRefreshTokenTTL(refreshTokenStr)
		blacklistKey := refreshTokenBlacklistPrefix + commonutil.HashSHA256(refreshTokenStr)
		if err := h.Redis.Set(ctx, blacklistKey, "1", ttl).Err(); err != nil {
			pkg.Logger.Warn("[Logout] Redis refresh token blacklist set failed", zap.Error(err))
		}
	}

	// 清除 Cookie
	clearAuthCookie(c)
	clearRefreshCookie(c)
	clearCSRFCookie(c)

	common.WriteJSON(c, http.StatusOK, common.OkData("登出成功"))
}

// RefreshAccessToken 使用 refresh token 刷新 access token
// POST /api/auth/refresh
func (h *EmployeeHandler) RefreshAccessToken(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	refreshTokenStr, _ := c.Cookie("refresh_token")
	if refreshTokenStr == "" {
		common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "缺少 refresh token"))
		return
	}

	// 检查 refresh token 是否已被撤销
	if h.Redis != nil {
		blacklistKey := refreshTokenBlacklistPrefix + commonutil.HashSHA256(refreshTokenStr)
		exists, err := h.Redis.Exists(ctx, blacklistKey).Result()
		if err != nil {
			pkg.Logger.Warn("[Refresh] Redis blacklist check failed", zap.Error(err))
		} else if exists > 0 {
			common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "refresh token 已被撤销"))
			return
		}
	}

	// 解析 refresh token 获取 employeeID
	employeeID, iatFloat, err := parseRefreshToken(refreshTokenStr)
	if err != nil {
		common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, err.Error()))
		return
	}

	// 原子锁防止并发刷新
	if h.Redis != nil {
		lockKey := fmt.Sprintf("%s%d", refreshLockPrefix, employeeID)
		locked, err := h.Redis.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
		if err != nil {
			pkg.Logger.Warn("[Refresh] Redis lock failed", zap.Error(err))
		} else if !locked {
			common.WriteJSON(c, http.StatusTooManyRequests, common.Fail(429, "请勿重复刷新token"))
			return
		}
		defer h.Redis.Del(context.Background(), lockKey)

		// 二次检查黑名单
		blacklistKey := refreshTokenBlacklistPrefix + commonutil.HashSHA256(refreshTokenStr)
		exists, err := h.Redis.Exists(ctx, blacklistKey).Result()
		if err == nil && exists > 0 {
			common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "refresh token 已被撤销"))
			return
		}

		// family-based 轮换检查
		familyKey := fmt.Sprintf("%s%d:%d", refreshFamilyPrefix, employeeID, int64(iatFloat))
		familyRevoked, err := h.Redis.Exists(ctx, familyKey).Result()
		if err == nil && familyRevoked > 0 {
			ttl := calcRefreshTokenTTL(refreshTokenStr)
			h.Redis.Set(ctx, refreshTokenBlacklistPrefix+commonutil.HashSHA256(refreshTokenStr), "1", ttl)
			common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "refresh token 已被撤销"))
			return
		}
	}

	// 调用 service 刷新
	accessToken, newRefreshToken, err := h.EmployeeService.RefreshToken(ctx, employeeID)
	if err != nil {
		common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "刷新token失败"))
		return
	}

	// 将旧 refresh token 加入黑名单
	if h.Redis != nil && refreshTokenStr != "" {
		ttl := calcRefreshTokenTTL(refreshTokenStr)
		blacklistKey := refreshTokenBlacklistPrefix + commonutil.HashSHA256(refreshTokenStr)
		_ = h.Redis.Set(ctx, blacklistKey, "1", ttl).Err()

		// family-based 撤销
		familyKey := fmt.Sprintf("%s%d:%d", refreshFamilyPrefix, employeeID, int64(iatFloat))
		_ = h.Redis.Set(ctx, familyKey, "1", ttl).Err()
	}

	setAuthCookie(c, accessToken)
	if newRefreshToken != "" {
		setRefreshCookie(c, newRefreshToken)
	}
	middleware.SetCSRFCookie(nil, c, accessToken)

	common.WriteJSON(c, http.StatusOK, common.OkData(map[string]string{"message": "token refreshed"}))
}

// ===================== 员工 CRUD =====================

// List 员工列表（分页查询）
// GET /api/employee/list
func (h *EmployeeHandler) List(c *gin.Context) {
	currentEmpID := middleware.GetEmployeeID(c)
	if !h.EmployeeService.IsAdmin(c.Request.Context(), currentEmpID) {
		common.WriteJSON(c, http.StatusForbidden, common.Fail(403, "无权执行此操作"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	realName := c.Query("realName")
	phone := c.Query("phone")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	req := &dto.EmployeeListRequest{
		RealName: realName,
		Phone:    phone,
		Page:     page,
		PageSize: pageSize,
	}

	resp, err := h.EmployeeService.ListEmployee(ctx, req)
	if err != nil {
		pkg.Logger.Error("ListEmployee failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "查询失败"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(resp))
}

// Create 添加员工
// POST /api/employee
func (h *EmployeeHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	var req dto.CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if req.RealName == "" {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "姓名不能为空"))
		return
	}
	if len(req.RealName) > model.MaxLenRealName {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "姓名长度不能超过50个字符"))
		return
	}
	if req.Phone != "" && !service.IsValidPhone(req.Phone) {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "手机号格式不正确"))
		return
	}
	if req.Email != "" && len(req.Email) > model.MaxLenEmail {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "邮箱长度不能超过100个字符"))
		return
	}
	if req.Email != "" && !isValidEmail(req.Email) {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "邮箱格式不正确"))
		return
	}
	if len(req.Department) > model.MaxLenDepartment {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "部门名称长度不能超过50个字符"))
		return
	}
	if len(req.Position) > model.MaxLenPosition {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "职位长度不能超过50个字符"))
		return
	}
	if len(req.Remark) > model.MaxLenRemark {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "备注长度不能超过200个字符"))
		return
	}

	emp, err := h.EmployeeService.CreateEmployee(ctx, &req)
	if err != nil {
		if strings.Contains(err.Error(), "已被使用") {
			common.WriteJSON(c, http.StatusConflict, common.Fail(409, "该手机号已被使用"))
			return
		}
		if errors.Is(err, service.ErrInvalidParam) {
			common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, err.Error()))
			return
		}
		pkg.Logger.Error("CreateEmployee failed", zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "创建失败"))
		return
	}

	common.WriteJSON(c, http.StatusCreated, common.OkData(emp))
}

// Update 更新员工
// PUT /api/employee/:id
func (h *EmployeeHandler) Update(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	currentEmpID := middleware.GetEmployeeID(c)

	idStr := c.Param("id")
	empID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "无效的员工ID"))
		return
	}

	// 仅允许更新本人信息
	if currentEmpID != empID {
		common.WriteJSON(c, http.StatusForbidden, common.Fail(403, "无权限执行此操作"))
		return
	}

	var req dto.UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if len(req.RealName) > model.MaxLenRealName {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "姓名长度不能超过50个字符"))
		return
	}
	if len(req.Remark) > model.MaxLenRemark {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "备注长度不能超过200个字符"))
		return
	}
	if len(req.Email) > model.MaxLenEmail {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "邮箱长度不能超过100个字符"))
		return
	}
	if req.Email != "" && !isValidEmail(req.Email) {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "邮箱格式不正确"))
		return
	}

	if err := h.EmployeeService.UpdateEmployee(ctx, empID, &req); err != nil {
		pkg.Logger.Error("UpdateEmployee failed", zap.Error(err), zap.Int64("id", empID))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "更新失败，请稍后重试"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(nil))
}

// ===================== 用户信息 =====================

// GetCurrentUser 获取当前登录用户信息
// GET /api/user/me
func (h *EmployeeHandler) GetCurrentUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	empID := middleware.GetEmployeeID(c)
	if empID == 0 {
		common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "未登录"))
		return
	}

	emp, err := h.EmployeeService.GetEmployeeByID(ctx, empID)
	if err != nil {
		pkg.Logger.Error("GetCurrentUser failed", zap.Int64("id", empID), zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "获取用户信息失败"))
		return
	}

	if emp == nil {
		common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "用户不存在"))
		return
	}

	userInfo := &dto.EmployeeInfo{
		ID:         emp.EmployeeID,
		RealName:   emp.RealName,
		Role:       emp.Role,
		Age:        emp.Age,
		Phone:      service.MaskPhone(emp.Phone),
		Email:      emp.Email,
		Avatar:     commonutil.DerefStr(emp.Avatar),
		Gender:     emp.Gender,
		Department: emp.Department,
		Position:   emp.Position,
		WorkStatus: emp.WorkStatus,
		CanLogin:   emp.CanLogin,
		CreateTime: emp.CreateTime,
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(userInfo))
}

// GetUserMenus 获取当前用户的菜单权限
// GET /api/user/menus
func (h *EmployeeHandler) GetUserMenus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	empID := middleware.GetEmployeeID(c)
	if empID == 0 {
		common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "未登录"))
		return
	}

	emp, err := h.EmployeeService.GetEmployeeByID(ctx, empID)
	if err != nil || emp == nil {
		common.WriteJSON(c, http.StatusNotFound, common.Fail(404, "用户不存在"))
		return
	}

	if h.RoleService == nil {
		common.WriteJSON(c, http.StatusOK, common.OkData(map[string]interface{}{
			"menus":       []interface{}{},
			"permissions": []interface{}{},
		}))
		return
	}

	menus, btnKeys, err := h.RoleService.GetUserMenus(ctx, emp.Role)
	if err != nil {
		pkg.Logger.Error("GetUserMenus failed", zap.Int64("id", empID), zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "获取菜单权限失败"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData(map[string]interface{}{
		"menus":       menus,
		"permissions": btnKeys,
	}))
}

// ===================== 密码管理 =====================

// ChangePassword 修改密码
// PUT /api/user/password
func (h *EmployeeHandler) ChangePassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	empID := middleware.GetEmployeeID(c)
	if empID == 0 {
		common.WriteJSON(c, http.StatusUnauthorized, common.Fail(401, "未登录"))
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "原密码和新密码不能为空"))
		return
	}

	if len(req.NewPassword) < model.PasswordMinLen || len(req.NewPassword) > model.PasswordMaxLen {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "新密码长度必须在8-64位之间"))
		return
	}

	if err := h.EmployeeService.ChangePassword(ctx, empID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrWrongOldPassword) {
			common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "原密码错误"))
			return
		}
		pkg.Logger.Error("ChangePassword failed", zap.Int64("id", empID), zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "密码修改失败，请稍后重试"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData("密码修改成功"))
}

// ResetPassword 管理员重置员工密码
// PUT /api/employee/:id/password
func (h *EmployeeHandler) ResetPassword(c *gin.Context) {
	currentEmpID := middleware.GetEmployeeID(c)
	if !h.EmployeeService.IsAdmin(c.Request.Context(), currentEmpID) {
		common.WriteJSON(c, http.StatusForbidden, common.Fail(403, "无权执行此操作"))
		return
	}

	idStr := c.Param("id")
	empID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || empID <= 0 {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "无效的员工ID"))
		return
	}

	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
		return
	}

	if req.NewPassword == "" {
		common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "新密码不能为空"))
		return
	}

	if err := h.EmployeeService.ResetPassword(c.Request.Context(), empID, req.NewPassword); err != nil {
		pkg.Logger.Error("ResetPassword failed", zap.Int64("id", empID), zap.Error(err))
		common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "密码重置失败"))
		return
	}

	common.WriteJSON(c, http.StatusOK, common.OkData("密码重置成功"))
}

// ===================== Cookie 辅助函数 =====================

func isSecureCookie() bool {
	return config.Cfg.App.Mode == "prod"
}

func setAuthCookie(c *gin.Context, token string) {
	expireHours := config.Cfg.App.JWT.ExpireHours
	if expireHours <= 0 {
		expireHours = 2
	}
	c.SetCookie("auth_token", token, expireHours*3600, "/", "", isSecureCookie(), true)
}

func setRefreshCookie(c *gin.Context, token string) {
	expireHours := config.Cfg.App.JWT.RefreshExpireHours
	if expireHours <= 0 {
		expireHours = 168
	}
	c.SetCookie("refresh_token", token, expireHours*3600, "/api/auth", "", isSecureCookie(), true)
}

func clearAuthCookie(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", isSecureCookie(), true)
}

func clearRefreshCookie(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/api/auth", "", isSecureCookie(), true)
}

func clearCSRFCookie(c *gin.Context) {
	c.SetCookie("xsrf-token", "", -1, "/", "", isSecureCookie(), false)
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	if len(email) < 5 || len(email) > 100 {
		return false
	}
	atIndex := -1
	dotIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false
			}
			atIndex = i
		}
		if c == '.' && atIndex != -1 {
			dotIndex = i
		}
	}
	return atIndex > 0 && dotIndex > atIndex+1 && dotIndex < len(email)-1
}

// ===================== Token 辅助函数 =====================

// parseRefreshToken 解析 refresh token，返回 employeeID 和 iat
func parseRefreshToken(tokenStr string) (int64, float64, error) {
	token, err := parseJWT(tokenStr)
	if err != nil || !token.Valid {
		return 0, 0, fmt.Errorf("refresh token 无效或已过期")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, 0, fmt.Errorf("refresh token 格式错误")
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return 0, 0, fmt.Errorf("token 类型错误")
	}

	employeeID, ok := claims["employee_id"].(float64)
	if !ok || employeeID == 0 {
		return 0, 0, fmt.Errorf("refresh token 无效")
	}

	iatFloat, _ := claims["iat"].(float64)
	return int64(employeeID), iatFloat, nil
}

// parseJWT 解析 JWT token
func parseJWT(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(service.GetJWTSecret()), nil
	})
}

// getAccessTokenTTL 计算 access token 剩余有效期作为黑名单 TTL
func getAccessTokenTTL(tokenStr string) time.Duration {
	t, err := parseJWT(tokenStr)
	if err != nil {
		return 2 * time.Hour
	}
	if claims, ok := t.Claims.(jwt.MapClaims); ok {
		if expFloat, ok := claims["exp"].(float64); ok {
			remaining := time.Until(time.Unix(int64(expFloat), 0))
			if remaining > 0 {
				return remaining
			}
		}
	}
	return 2 * time.Hour
}

// calcRefreshTokenTTL 计算 refresh token 剩余有效期
func calcRefreshTokenTTL(tokenStr string) time.Duration {
	ttl := 7 * 24 * time.Hour
	t, err := parseJWT(tokenStr)
	if err != nil {
		return ttl
	}
	if claims, ok := t.Claims.(jwt.MapClaims); ok {
		if expFloat, ok := claims["exp"].(float64); ok {
			remaining := time.Until(time.Unix(int64(expFloat), 0))
			if remaining > 0 && remaining < ttl {
				return remaining
			}
		}
	}
	return ttl
}
