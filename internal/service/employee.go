package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	commonutil "shopmall/common/util"
	"shopmall/internal/config"
	"shopmall/internal/dto"
	"shopmall/internal/model"
	"shopmall/internal/repository"
	"shopmall/pkg"
)

var (
	ErrInvalidCredentials = errors.New("手机号或密码错误")
	ErrEmployeeDisabled   = errors.New("账号已被禁用")
	ErrCannotLogin        = errors.New("该账号不允许登录后台")
	ErrInvalidParam       = errors.New("参数无效")
	ErrWrongOldPassword   = errors.New("原密码错误")
)

// JWTSecret JWT密钥（从环境变量读取）
var JWTSecret string

// GetJWTSecret 延迟获取 JWT Secret，确保在 config.Load() 加载 config.yaml 之后调用
func GetJWTSecret() string {
	if JWTSecret != "" {
		return JWTSecret
	}
	secret := config.Cfg.App.JWTSecret
	if secret == "" {
		log.Fatal("app.jwt_secret 未配置，请在 config.yaml 中设置（至少32字符）")
	}
	if len(secret) < 32 && config.Cfg.App.Mode == "prod" {
		log.Fatal("app.jwt_secret 长度必须至少32字符（当前生产环境）")
	}
	JWTSecret = secret
	return JWTSecret
}

type EmployeeService struct {
	employeeRepo *repository.EmployeeRepo
	redis        *redis.Client // 密码修改后使旧 token 失效
}

func NewEmployeeSvc(repo *repository.EmployeeRepo, redisClient *redis.Client) *EmployeeService {
	return &EmployeeService{employeeRepo: repo, redis: redisClient}
}

// invalidateUserTokens 密码修改/重置后，记录密码变更时间戳使旧 token 失效
// 在 Redis 中写入 pw_version:{empID} = Unix 时间戳，TTL 动态计算
func (s *EmployeeService) invalidateUserTokens(ctx context.Context, empID int64) {
	if s.redis == nil {
		return
	}
	// 动态 TTL：取 access token 过期时间 + 5 分钟缓冲，至少 150 分钟
	ttl := 150 * time.Minute
	if h := config.Cfg.App.JWT.ExpireHours; h > 0 {
		dynamicTTL := time.Duration(h)*time.Hour + 5*time.Minute
		if dynamicTTL > ttl {
			ttl = dynamicTTL
		}
	}
	key := fmt.Sprintf("pw_version:%d", empID)
	if err := s.redis.Set(ctx, key, time.Now().Unix(), ttl).Err(); err != nil {
		pkg.Logger.Warn("[EmployeeService] invalidateUserTokens Redis set failed", zap.Error(err))
	}
}

// CheckPasswordVersion 检查 JWT 签发时间是否早于密码修改时间
// 返回 true 表示 token 仍然有效，false 表示密码已修改需要重新登录
func (s *EmployeeService) CheckPasswordVersion(ctx context.Context, empID int64, iat int64) bool {
	if s.redis == nil {
		return true // 无 Redis 时不检查（单实例部署）
	}
	key := fmt.Sprintf("pw_version:%d", empID)
	val, err := s.redis.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			// key 不存在 = 密码从未修改过，token 有效
			return true
		}
		// Redis 连接异常时 fail-close：要求重新登录
		pkg.Logger.Error("[Security] pw_version Redis check failed, rejecting token",
			zap.Int64("empID", empID), zap.Error(err))
		return false
	}
	return iat >= val // token 签发时间 >= 密码修改时间则有效
}

// IsAdmin 检查员工是否为管理员角色
func (s *EmployeeService) IsAdmin(ctx context.Context, empID int64) bool {
	emp, err := s.employeeRepo.GetByIDWithoutPassword(ctx, empID)
	if err != nil || emp == nil {
		return false
	}
	return emp.Role == "super_admin" || emp.Role == "admin"
}

// Login 员工登录
func (s *EmployeeService) Login(ctx context.Context, req *dto.LoginRequest, loginIP string) (*dto.LoginResponse, string, string, error) {
	emp, err := s.employeeRepo.GetByPhone(ctx, req.Phone)
	if err != nil || emp == nil {
		s.writeLoginLog(ctx, nil, req.Phone, loginIP, 0, "手机号不存在")
		return nil, "", "", ErrInvalidCredentials
	}

	if emp.CanLogin == model.CanLoginNo {
		s.writeLoginLog(ctx, &emp.EmployeeID, req.Phone, loginIP, 0, "该账号不允许登录后台")
		return nil, "", "", ErrCannotLogin
	}

	if emp.WorkStatus == model.WorkStatusDimission {
		s.writeLoginLog(ctx, &emp.EmployeeID, req.Phone, loginIP, 0, "账号已离职")
		return nil, "", "", ErrEmployeeDisabled
	}

	if !verifyPassword(req.Password, commonutil.DerefStr(emp.Password)) {
		s.writeLoginLog(ctx, &emp.EmployeeID, req.Phone, loginIP, 0, "密码错误")
		return nil, "", "", ErrInvalidCredentials
	}

	token, err := generateToken(emp.EmployeeID)
	if err != nil {
		pkg.Logger.Error("生成Token失败", zap.Error(err))
		return nil, "", "", errors.New("登录失败，请重试")
	}

	refreshToken, err := generateRefreshToken(emp.EmployeeID)
	if err != nil {
		pkg.Logger.Error("生成RefreshToken失败", zap.Error(err))
		return nil, "", "", errors.New("登录失败，请重试")
	}

	s.writeLoginLog(ctx, &emp.EmployeeID, req.Phone, loginIP, 1, "登录成功")

	return &dto.LoginResponse{
		EmployeeInfo: &dto.EmployeeInfo{
			ID:         emp.EmployeeID,
			RealName:   emp.RealName,
			Role:       emp.Role,
			Age:        emp.Age,
			Phone:      MaskPhone(emp.Phone),
			Email:      emp.Email,
			Avatar:     commonutil.DerefStr(emp.Avatar),
			Gender:     emp.Gender,
			Department: emp.Department,
			Position:   emp.Position,
			WorkStatus: emp.WorkStatus,
			CanLogin:   emp.CanLogin,
			CreateTime: emp.CreateTime,
		},
	}, token, refreshToken, nil
}

// writeLoginLog 记录登录日志（手机号脱敏处理）
func (s *EmployeeService) writeLoginLog(ctx context.Context, employeeID *int64, phone, ip string, status int8, msg string) {
	log := &model.SysLoginLog{
		EmployeeID: employeeID,
		Phone:      MaskPhone(phone), // P1: 手机号脱敏后再写入日志
		IP:         ip,
		Status:     status,
		Msg:        msg,
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				pkg.Logger.Error("[LoginLog] 写入登录日志 panic", zap.Any("panic", r))
			}
		}()
		if err := s.employeeRepo.InsertLoginLog(context.Background(), log); err != nil {
			pkg.Logger.Error("[LoginLog] 写入登录日志失败", zap.Error(err))
		}
	}()
}

// MaskPhone 手机号脱敏：保留前3位和后4位，中间用*替代
func MaskPhone(phone string) string {
	n := len(phone)
	if n <= 7 {
		return phone
	}
	return phone[:3] + "****" + phone[n-4:]
}

// verifyPassword 验证密码（bcrypt）
func verifyPassword(input, stored string) bool {
	if stored == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(input)) == nil
}

// HashPassword 生成 bcrypt 密码哈希（用于创建/修改密码），cost=12 符合 OWASP 建议
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

func IsValidPhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}

// generateToken 生成 JWT Token（access token，短有效期）
func generateToken(employeeID int64) (string, error) {
	now := time.Now()
	expireHours := config.Cfg.App.JWT.ExpireHours
	claims := jwt.MapClaims{
		"employee_id": employeeID,
		"iat":         now.Unix(),
		"exp":         now.Add(time.Duration(expireHours) * time.Hour).Unix(),
		"type":        "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(GetJWTSecret()))
}

// generateRefreshToken 生成 Refresh Token（长有效期，仅用于换 access token）
func generateRefreshToken(employeeID int64) (string, error) {
	now := time.Now()
	expireHours := config.Cfg.App.JWT.RefreshExpireHours
	claims := jwt.MapClaims{
		"employee_id": employeeID,
		"iat":         now.Unix(),
		"exp":         now.Add(time.Duration(expireHours) * time.Hour).Unix(),
		"type":        "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(GetJWTSecret()))
}

// RefreshToken 使用 refresh token 换取新的 access token + 轮换 refresh token
func (s *EmployeeService) RefreshToken(ctx context.Context, employeeID int64) (accessToken, refreshToken string, err error) {
	// 验证员工仍然有效
	emp, err := s.employeeRepo.GetByIDWithoutPassword(ctx, employeeID)
	if err != nil || emp == nil {
		return "", "", errors.New("用户不存在")
	}
	if emp.CanLogin != model.CanLoginYes || emp.WorkStatus != model.WorkStatusOnJob {
		return "", "", errors.New("账号已被禁用或离职")
	}

	accessToken, err = generateToken(employeeID)
	if err != nil {
		pkg.Logger.Error("RefreshToken: generateToken failed", zap.Error(err))
		return "", "", errors.New("刷新令牌失败")
	}
	refreshToken, err = generateRefreshToken(employeeID)
	if err != nil {
		pkg.Logger.Error("RefreshToken: generateRefreshToken failed", zap.Error(err))
		return "", "", errors.New("刷新令牌失败")
	}
	return accessToken, refreshToken, nil
}

// ListEmployee 分页查询员工列表
func (s *EmployeeService) ListEmployee(ctx context.Context, req *dto.EmployeeListRequest) (*dto.EmployeeListResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > model.MaxPageSize {
		req.PageSize = model.DefaultPageSize
	}

	list, total, err := s.employeeRepo.ListEmployee(ctx, req.RealName, req.Phone, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.EmployeeInfo, 0, len(list))
	for _, emp := range list {
		items = append(items, &dto.EmployeeInfo{
			ID:         emp.EmployeeID,
			RealName:   emp.RealName,
			Role:       emp.Role,
			Age:        emp.Age,
			Phone:      MaskPhone(emp.Phone), // PII: 列表脱敏
			Email:      emp.Email,
			Avatar:     commonutil.DerefStr(emp.Avatar),
			Gender:     emp.Gender,
			Department: emp.Department,
			Position:   emp.Position,
			WorkStatus: emp.WorkStatus,
			CanLogin:   emp.CanLogin,
			CreateTime: emp.CreateTime,
		})
	}

	return &dto.EmployeeListResponse{
		List:     items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// CreateEmployee 创建员工（不再处理 role 字段）
func (s *EmployeeService) CreateEmployee(ctx context.Context, req *dto.CreateEmployeeRequest) (*dto.EmployeeInfo, error) {
	if req.RealName == "" {
		return nil, errors.New("姓名不能为空")
	}
	if req.Phone == "" {
		return nil, errors.New("手机号不能为空")
	}
	if !IsValidPhone(req.Phone) {
		return nil, errors.New("手机号格式不正确")
	}

	if existing, dupErr := s.employeeRepo.GetByPhone(ctx, req.Phone); dupErr == nil && existing != nil {
		return nil, fmt.Errorf("手机号 %s 已被使用", req.Phone)
	}

	// 枚举值校验（不含 role）
	if req.Gender == 0 {
		req.Gender = int8(model.GenderUnknown)
	}
	if req.Gender < int8(model.GenderUnknown) || req.Gender > int8(model.GenderOther) {
		return nil, errors.New("性别值无效，应为 0(未知) 1(男) 2(女) 3(其他)")
	}
	if req.WorkStatus == 0 {
		req.WorkStatus = int8(model.WorkStatusOnJob)
	}
	if req.WorkStatus != int8(model.WorkStatusDimission) && req.WorkStatus != int8(model.WorkStatusOnJob) {
		return nil, errors.New("在职状态值无效，应为 0(离职) 或 1(在职)")
	}
	if req.CanLogin == 0 {
		req.CanLogin = int8(model.CanLoginYes)
	}
	if req.CanLogin != int8(model.CanLoginNo) && req.CanLogin != int8(model.CanLoginYes) {
		return nil, errors.New("是否允许登录值无效，应为 0(否) 或 1(是)")
	}

	// 密码处理：创建员工时必须提供密码
	var passwordHash string
	var err error
	if req.Password == "" {
		return nil, errors.New("创建员工必须设置密码")
	}
	if len(req.Password) < model.PasswordMinLen || len(req.Password) > model.PasswordMaxLen {
		return nil, errors.New("密码长度必须在8-64位之间")
	}
	// 密码复杂度校验（大写+小写+数字+特殊字符）
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range req.Password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case c >= 0x21 && c <= 0x2F || c >= 0x3A && c <= 0x40 || c >= 0x5B && c <= 0x60 || c >= 0x7B && c <= 0x7E:
			hasSpecial = true
		}
	}
	if !(hasUpper && hasLower && hasDigit && hasSpecial) {
		return nil, errors.New("密码必须包含大写字母、小写字母、数字和特殊字符")
	}
	passwordHash, err = HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	var avatar *string
	if req.Avatar != "" {
		avatar = &req.Avatar
	}
	remark := req.Remark

	emp := &model.SysEmployee{
		RealName:   req.RealName,
		Phone:      req.Phone,
		Password:   &passwordHash,
		Avatar:     avatar,
		Gender:     req.Gender,
		Age:        req.Age,
		Email:      req.Email,
		Department: req.Department,
		Position:   req.Position,
		WorkStatus: model.WorkStatusType(req.WorkStatus),
		CanLogin:   model.CanLoginType(req.CanLogin),
		Remark:     remark,
	}

	if err := s.employeeRepo.InsertEmployee(ctx, emp); err != nil {
		return nil, err
	}

	return &dto.EmployeeInfo{
		ID:         emp.EmployeeID,
		RealName:   emp.RealName,
		Role:       emp.Role,
		Age:        emp.Age,
		Phone:      MaskPhone(emp.Phone),
		Email:      emp.Email,
		Avatar:     commonutil.DerefStr(emp.Avatar),
		Gender:     emp.Gender,
		Department: emp.Department,
		Position:   emp.Position,
		WorkStatus: emp.WorkStatus,
		CanLogin:   emp.CanLogin,
		CreateTime: emp.CreateTime,
	}, nil
}

// UpdateEmployee 更新员工（去掉 callerRole，仅更新传入的合法字段）
func (s *EmployeeService) UpdateEmployee(ctx context.Context, id int64, req *dto.UpdateEmployeeRequest) error {
	if req.RealName == "" {
		return errors.New("姓名不能为空")
	}

	existing, err := s.employeeRepo.GetByID(ctx, id)
	if err != nil || existing == nil {
		return errors.New("员工不存在")
	}

	// 枚举值校验
	if req.Gender < int8(model.GenderUnknown) || req.Gender > int8(model.GenderOther) {
		return errors.New("性别值无效，应为 0(未知) 1(男) 2(女) 3(其他)")
	}

	// 更新字段（仅允许更新非敏感字段）
	existing.RealName = req.RealName
	if req.Phone != "" {
		if !IsValidPhone(req.Phone) {
			return errors.New("手机号格式不正确")
		}
		// 检查手机号是否被其他员工使用
		if existing.Phone != req.Phone {
			if dup, dupErr := s.employeeRepo.GetByPhone(ctx, req.Phone); dupErr == nil && dup != nil && dup.EmployeeID != id {
				return fmt.Errorf("手机号 %s 已被其他员工使用", req.Phone)
			}
		}
		existing.Phone = req.Phone
	}
	if req.Avatar != "" {
		existing.Avatar = &req.Avatar
	}
	existing.Gender = req.Gender
	if req.Age != nil {
		existing.Age = req.Age
	}
	if req.Email != "" {
		existing.Email = req.Email
	}
	if req.Remark != "" {
		existing.Remark = req.Remark
	}
	// P1: 以下敏感字段不允许普通用户自行修改
	// department, position, work_status, can_login 应由管理员接口修改

	return s.employeeRepo.UpdateEmployee(ctx, existing)
}

// GetEmployeeByID 根据 ID 获取员工信息（不含密码，用于 /user/me）
func (s *EmployeeService) GetEmployeeByID(ctx context.Context, id int64) (*model.SysEmployee, error) {
	return s.employeeRepo.GetByIDWithoutPassword(ctx, id)
}

// ChangePassword 修改密码
func (s *EmployeeService) ChangePassword(ctx context.Context, empID int64, oldPassword, newPassword string) error {
	emp, err := s.employeeRepo.GetByID(ctx, empID)
	if err != nil || emp == nil {
		return errors.New("员工不存在")
	}

	if emp.Password == nil || *emp.Password == "" {
		return errors.New("账号密码异常，请联系管理员")
	}

	if !verifyPassword(oldPassword, *emp.Password) {
		return ErrWrongOldPassword
	}

	if len(newPassword) < model.PasswordMinLen || len(newPassword) > model.PasswordMaxLen {
		return fmt.Errorf("密码长度必须在%d-%d位之间", model.PasswordMinLen, model.PasswordMaxLen)
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range newPassword {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case c >= 0x21 && c <= 0x2F || c >= 0x3A && c <= 0x40 || c >= 0x5B && c <= 0x60 || c >= 0x7B && c <= 0x7E:
			hasSpecial = true
		}
	}
	if !(hasUpper && hasLower && hasDigit && hasSpecial) {
		return errors.New("密码必须包含大写字母、小写字母、数字和特殊字符")
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return errors.New("密码加密失败")
	}

	if err := s.employeeRepo.UpdatePassword(ctx, empID, hash); err != nil {
		pkg.Logger.Error("ChangePassword: update failed", zap.Int64("id", empID), zap.Error(err))
		return errors.New("密码修改失败")
	}

	s.invalidateUserTokens(ctx, empID)
	return nil
}

// ResetPassword 管理员重置员工密码（不需要旧密码）
func (s *EmployeeService) ResetPassword(ctx context.Context, empID int64, newPassword string) error {
	emp, err := s.employeeRepo.GetByID(ctx, empID)
	if err != nil || emp == nil {
		return errors.New("员工不存在")
	}

	if len(newPassword) < model.PasswordMinLen || len(newPassword) > model.PasswordMaxLen {
		return fmt.Errorf("密码长度必须在%d-%d位之间", model.PasswordMinLen, model.PasswordMaxLen)
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range newPassword {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case c >= 0x21 && c <= 0x2F || c >= 0x3A && c <= 0x40 || c >= 0x5B && c <= 0x60 || c >= 0x7B && c <= 0x7E:
			hasSpecial = true
		}
	}
	if !(hasUpper && hasLower && hasDigit && hasSpecial) {
		return errors.New("密码必须包含大写字母、小写字母、数字和特殊字符")
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return errors.New("密码加密失败")
	}

	if err := s.employeeRepo.UpdatePassword(ctx, empID, hash); err != nil {
		pkg.Logger.Error("ResetPassword: update failed", zap.Int64("id", empID), zap.Error(err))
		return errors.New("密码重置失败")
	}

	s.invalidateUserTokens(ctx, empID)
	return nil
}
