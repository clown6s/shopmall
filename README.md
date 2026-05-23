# ShopMall

基于 Go + Gin 的电商后台管理系统，采用 Repository-Service-Handler 分层架构，Wire 编译期依赖注入。

## 技术栈

| 组件 | 选型 |
|------|------|
| HTTP 框架 | Gin v1.12 |
| 数据库 | MySQL + database/sql |
| 缓存 | Redis (go-redis/v9) |
| 依赖注入 | Wire v0.7 |
| 日志 | Zap |
| 配置 | YAML |

## 项目结构

```
shopmall/
├── main.go                      # 入口：加载配置 → Wire 注入 → 启动 HTTP 服务
├── config.yaml                  # 配置文件
├── config.yaml.example          # 配置模板
├── cmd/server/
│   ├── wire.go                  # Wire 依赖声明（wireinject tag）
│   └── wire_gen.go              # Wire 生成的代码（勿手动修改）
├── internal/
│   ├── config/config.go         # 配置结构体 + YAML 加载 + 校验
│   ├── handler/
│   │   ├── handler.go       # Handler 容器
│   │   └── employee.go      # 员工 Handler（登录/增删改查/密码管理）
│   ├── model/
│   │   ├── constants.go     # 分页/密码/字段长度常量
│   │   ├── product.go       # 商品相关模型
│   │   └── sys_permission.go# 系统模型（员工/部门/登录日志）
│   ├── repository/
│   │   └── employee.go      # 员工数据访问层
│   ├── service/
│   │   ├── employee.go      # 员工业务逻辑
│   │   └── role.go          # 角色服务（桩实现）
│   └── dto/
│       └── employee.go      # 员工请求/响应 DTO
├── common/
│   ├── response.go              # 统一响应 R{Code, Message, Data, TraceID}
│   ├── util/util.go             # 工具函数（SHA256、安全解引用）
│   └── middleware/              # 中间件
│       ├── trace.go             # X-Trace-ID 请求追踪
│       ├── cors.go              # CORS 跨域
│       ├── recovery.go          # Panic 恢复
│       ├── security_headers.go  # 安全响应头（CSP/HSTS/X-Frame-Options）
│       ├── body_limit.go        # 请求体大小限制
│       ├── context.go           # 请求上下文超时
│       ├── jwt.go               # JWT 认证辅助函数
│       ├── csrf.go              # CSRF Cookie（桩实现）
│       └── rate_limiter.go      # 登录速率限制（桩实现）
├── db/mysql.go                  # MySQL 连接池初始化
├── cache/redis.go               # Redis 初始化
├── pkg/logger.go                # Zap 日志（dev/prod 模式）
└── routes/routes.go             # 路由注册
```

## 快速开始

```bash
# 1. 复制配置文件
cp config.yaml.example config.yaml
# 编辑 config.yaml，填入 MySQL/Redis 密码

# 2. 运行
go run main.go

# 3. 指定配置文件路径
go run main.go -config /path/to/config.yaml
```

服务启动后访问 `http://localhost:7000/health` 检查健康状态。

## 配置说明

配置文件为 `config.yaml`，字段说明：

```yaml
app:
  port: "7000"        # 监听端口
  mode: dev           # dev | prod | test

db:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ""        # 生产模式必填
  dbname: shop_mall
  max_open_conns: 50
  max_idle_conns: 25
  conn_max_life: 3600        # 连接最大存活时间（秒）
  conn_max_idle_time: 600    # 空闲连接最大存活时间（秒）
  tls: ""                    # "" | skip-verify | 自定义

redis:
  host: 127.0.0.1
  port: 6379
  password: ""        # 生产模式必填
  db: 0
  tls: false

cors:
  allow_origins:
    - http://localhost:5173
  allow_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  allow_headers:
    - Origin
    - Content-Type
    - Authorization
    - X-XSRF-TOKEN
  allow_credentials: true
  max_age: 86400
```

所有字段都有默认值，未填写的字段会使用内置默认值。生产模式下会强制要求 MySQL 和 Redis 密码。

## Docker 部署

Docker 部署时通过 volume mount 覆盖配置文件即可，无需重新编译：

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o shopmall main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/shopmall /shopmall
EXPOSE 7000
ENTRYPOINT ["/shopmall"]
```

```bash
# 构建镜像
docker build -t shopmall:latest .

# 运行，挂载生产配置
docker run -d \
  -p 7000:7000 \
  -v /etc/shopmall/config.yaml:/app/config.yaml:ro \
  shopmall:latest \
  -config /app/config.yaml
```

生产环境 `config.yaml` 示例：

```yaml
app:
  port: "7000"
  mode: prod

db:
  host: mysql
  port: 3306
  user: shopmall
  password: "your-strong-password"
  dbname: shop_mall
  max_open_conns: 100
  max_idle_conns: 50

redis:
  host: redis
  port: 6379
  password: "your-redis-password"

cors:
  allow_origins:
    - https://admin.yourdomain.com
```

---

## Wire 依赖注入教程

### 什么是 Wire

[Wire](https://github.com/google/wire) 是 Google 开源的**编译期**依赖注入工具。和运行时 DI（如 Uber Dig）不同，Wire 在编译阶段生成依赖注入代码，零运行时开销，编译时就能发现依赖缺失。

### 核心概念

| 概念 | 说明 |
|------|------|
| **Provider** | 一个普通函数，参数是它的依赖，返回值是它创建的实例。Wire 自动调用它 |
| **Provider Set** | 一组 Provider 的集合，按层级组织，方便复用 |
| **Injector** | 入口函数，声明"外部给我什么参数，Wire 帮我组装出什么结果" |
| **wire.Build** | 在 Injector 里列出所有 Provider，Wire 根据类型匹配自动推导调用顺序 |

### 安装 Wire

```bash
go install github.com/google/wire/cmd/wire@latest
```

### Wire 怎么工作

Wire 的核心逻辑就一句话：**匹配函数签名的类型**。

```
NewEmployeeRepo(db *sql.DB) *EmployeeRepo
                 ^^^^^^^^              ^^^^^^^^^^^^^^^
              Wire 去找谁能         Wire 知道这个函数
              提供这个类型           能提供这个类型
```

当 Wire 看到某个函数需要 `*sql.DB`，它就去找所有 Provider 的返回值，找到 `InitMySQL` 返回 `*sql.DB`，于是自动调用 `InitMySQL` 创建实例，再传给 `NewEmployeeRepo`。整个过程是**编译期推导**，不需要运行时反射。

### 项目 Wire 文件说明

```
cmd/server/
├── wire.go       # 你手动编辑的文件，声明 Provider 和 Injector
└── wire_gen.go   # Wire 自动生成的文件，不要手动编辑
```

- **`wire.go`** — 你唯一需要编辑的文件。顶部有 `//go:build wireinject` 标签，Go 编译器会忽略它
- **`wire_gen.go`** — 每次 `wire` 命令会完全覆盖这个文件

### 当前 wire.go 完整内容

```go
//go:build wireinject
// +build wireinject

package server

import (
    "database/sql"

    "shopmall/cache"
    "shopmall/common/middleware"
    "shopmall/db"
    "shopmall/internal/config"
    "shopmall/internal/handler"
    "shopmall/internal/repository"
    "shopmall/internal/service"

    "github.com/google/wire"
    "github.com/redis/go-redis/v9"
)

// App 应用依赖容器，所有顶层依赖集中在这里
type App struct {
    DB              *sql.DB
    Redis           *redis.Client
    Cfg             *config.Config
    EmployeeHandler *handler.EmployeeHandler
}

// newApp 组装函数，Wire 自动调用
func newApp(
    db *sql.DB,
    rdb *redis.Client,
    cfg *config.Config,
    empHandler *handler.EmployeeHandler,
) *App {
    return &App{
        DB:              db,
        Redis:           rdb,
        Cfg:             cfg,
        EmployeeHandler: empHandler,
    }
}

// wireSet 所有 Provider 按层组织
var wireSet = wire.NewSet(
    // 基础设施
    db.InitMySQL,
    cache.InitRedis,

    // Repository 层
    repository.NewEmployeeRepo,

    // Service 层
    service.NewEmployeeSvc,
    service.NewRoleSvc,

    // Middleware
    middleware.NewLoginRateLimiter,

    // Handler 层
    handler.NewEmployeeHandler,

    // App 组装
    newApp,
)

// InitializeApp Wire 入口（Injector）
// 参数：外部传入的依赖（Config 由 main.go 加载后传入）
// 返回：Wire 自动组装的 App
func InitializeApp(cfg *config.Config) (*App, error) {
    wire.Build(wireSet)
    return &App{}, nil  // 占位，Wire 会重写整个函数
}
```

### 当前依赖关系图

```
main.go 传入 *config.Config
    │
    ├── db.InitMySQL(&cfg.DB) ─────────────────────→ *sql.DB
    │                                                   │
    ├── cache.InitRedis(&cfg.Redis) ────────────────→ *redis.Client
    │                                                   │
    │   ┌───────────────────────────────────────────────┘
    │   │
    │   ├── repository.NewEmployeeRepo(db) ─────────→ *repository.EmployeeRepo
    │   │                                                  │
    │   └── service.NewEmployeeSvc(repo, redis) ─────→ *service.EmployeeService
    │                                                      │
    │   service.NewRoleSvc() ────────────────────────→ *service.RoleService ──────┐
    │                                                                           │
    │   middleware.NewLoginRateLimiter() ─────────────→ *middleware.LoginRateLimiter ─┤
    │                                                                               │
    │   └───────────────────────────────────────────────────────────────────────────┘
    │                                                                               │
    └── handler.NewEmployeeHandler(svc, roleSvc, limiter, redis) ──→ *handler.EmployeeHandler
                                                                                     │
    newApp(db, redis, cfg, empHandler) ─────────────────────────────────────────→ *App
```

### 添加新业务模块的步骤

假设要添加一个「商品管理」模块，完整的 Wire 操作流程：

#### 第 1 步：编写 Provider 函数（每层一个）

```go
// internal/repository/product.go
func NewProductRepo(db *sql.DB) *ProductRepo {
    return &ProductRepo{db: db}
}
```

```go
// internal/service/product.go
func NewProductSvc(repo *ProductRepo, redis *redis.Client) *ProductService {
    return &ProductService{productRepo: repo, redis: redis}
}
```

```go
// internal/handler/product.go
func NewProductHandler(svc *ProductService) *ProductHandler {
    return &ProductHandler{ProductService: svc}
}
```

**关键**：Provider 函数的参数类型必须是其他 Provider 能提供的类型。Wire 通过类型匹配自动推导调用顺序。

#### 第 2 步：把新 Handler 加到 App 容器

```go
// cmd/server/wire.go

type App struct {
    DB              *sql.DB
    Redis           *redis.Client
    Cfg             *config.Config
    EmployeeHandler *handler.EmployeeHandler
    ProductHandler  *handler.ProductHandler  // 新增
}
```

#### 第 3 步：更新 newApp 函数签名

```go
func newApp(
    db *sql.DB,
    rdb *redis.Client,
    cfg *config.Config,
    empHandler *handler.EmployeeHandler,
    prodHandler *handler.ProductHandler,  // 新增
) *App {
    return &App{
        DB:              db,
        Redis:           rdb,
        Cfg:             cfg,
        EmployeeHandler: empHandler,
        ProductHandler:  prodHandler,  // 新增
    }
}
```

#### 第 4 步：在 wireSet 中注册 Provider

```go
var wireSet = wire.NewSet(
    // 基础设施
    db.InitMySQL,
    cache.InitRedis,

    // Repository 层
    repository.NewEmployeeRepo,
    repository.NewProductRepo,    // 新增

    // Service 层
    service.NewEmployeeSvc,
    service.NewRoleSvc,
    service.NewProductSvc,        // 新增

    // Middleware
    middleware.NewLoginRateLimiter,

    // Handler 层
    handler.NewEmployeeHandler,
    handler.NewProductHandler,    // 新增

    // App 组装
    newApp,
)
```

**注意**：wireSet 中的顺序不影响 Wire 推导，Wire 纯靠类型匹配。但按层分组写方便阅读和维护。

#### 第 5 步：添加 import（别忘了）

```go
import (
    // ... 已有 import
    "shopmall/internal/repository"
    "shopmall/internal/service"
    "shopmall/internal/handler"
    // Wire 会自动推导，不需要 import 具体的 Repo/Svc 类型
)
```

如果新模块用了新的包，确保 import 路径正确。

#### 第 6 步：生成 wire_gen.go

```bash
cd cmd/server && wire
```

Wire 会自动分析 wireSet 中所有 Provider 的函数签名，推导依赖链，生成 `wire_gen.go`。

#### 第 7 步：验证编译

```bash
go build ./...
```

编译通过说明依赖注入配置正确。

### 添加新模块 Checklist

添加新业务模块到 Wire 时，按此清单逐项完成：

- [ ] 每层编写 `NewXxx` Provider 函数（Repository / Service / Handler）
- [ ] 在 `App` 结构体中添加新 Handler 字段
- [ ] 更新 `newApp` 函数签名，添加新参数
- [ ] 更新 `newApp` 函数体，赋值新字段
- [ ] 在 `wireSet` 中注册新的 Provider 函数（按层分组）
- [ ] 确认 import 路径正确
- [ ] `cd cmd/server && wire` 重新生成
- [ ] `go build ./...` 验证编译通过
- [ ] 在 `main.go` 中把新 Handler 传给路由注册函数

### Wire 常用 API

```go
// Provider Set：组织多个 Provider（按层分组）
wire.NewSet(
    repository.NewEmployeeRepo,
    repository.NewProductRepo,
)

// 结构体字段自动注入：Wire 自动匹配类型注入到指定字段
var Set = wire.NewSet(
    wire.Struct(new(App), "DB", "Redis"), // 只注入 DB 和 Redis 字段
    // wire.Struct(new(App))               // 注入所有匹配字段
)

// 绑定接口到实现（当有接口时用）
var Set = wire.NewSet(
    wire.Bind(new(UserRepository), new(*userRepositoryImpl)),
    NewUserRepoImpl,
)

// 值 Provider：直接注入一个固定值
var Set = wire.NewSet(
    wire.Value("default-secret-key"), // 提供 string 类型
)

// 清理函数：依赖创建失败时自动调用
func InitMySQL(cfg *config.DBConfig) (*sql.DB, func(), error) {
    db, _ := sql.Open("mysql", dsn)
    cleanup := func() { db.Close() }
    return db, cleanup, nil
}
```

### go generate 自动生成

推荐用 `go generate` 触发 Wire，这样不用记 `cd` 到哪个目录：

```bash
# 在任意 go 文件顶部加这行注释
//go:generate go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server

# 然后执行
go generate ./...
```

### 注意事项

1. **`wire_gen.go` 不要手动编辑**，每次 `wire` 命令会完全覆盖
2. **`wire.go` 顶部必须有 `//go:build wireinject`**，否则编译器会同时编译两个文件导致重复定义
3. **循环依赖**：Wire 不支持循环依赖，如果出现说明架构设计有问题，需要重构
4. **同一类型多个 Provider**：使用 `wire.Bind` 绑定接口来区分
5. **Provider 函数签名是核心**：Wire 只看参数类型和返回值类型，函数名不影响推导

---

## 业务模块开发教程

本节以 **「添加员工」** 功能为完整示例，演示在 ShopMall 中开发一个业务模块（数据写入操作）的全流程。

### 架构总览

```
请求 → Route → Handler → Service → Repository → MySQL
         ↓        ↓          ↓           ↓
      路由分发   参数校验    业务逻辑    数据访问
```

每一层只做自己的事，不越权。Handler 不写 SQL，Repository 不做业务校验，Service 不操作 HTTP。

### 开发步骤

一个完整的 Write 操作需要依次创建以下文件：

| 步骤 | 层 | 文件 | 职责 |
|------|----|------|------|
| 1 | Model | `internal/model/sys_permission.go` | 数据表结构定义 |
| 2 | DTO | `internal/dto/employee.go` | 请求/响应结构体 |
| 3 | Repository | `internal/repository/employee.go` | SQL INSERT/UPDATE |
| 4 | Service | `internal/service/employee.go` | 校验、加密、调用 Repo |
| 5 | Handler | `internal/handler/employee.go` | 参数解析、调用 Service、返回 JSON |
| 6 | Route | `routes/routes.go` | 注册路由 |
| 7 | Wire | `cmd/server/wire.go` | 依赖注入串联 |

---

### 第 1 步：Model（数据模型）

Model 与数据库表一一对应，定义字段类型和 JSON 标签。

```go
// internal/model/sys_permission.go

package model

import "time"

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

// SysEmployee 员工表
type SysEmployee struct {
    EmployeeID int64          `json:"employeeId"`
    Password   *string        `json:"-"`         // 加密后的密码，禁止 JSON 序列化
    RealName   string         `json:"realName"`
    Avatar     *string        `json:"avatar"`
    Gender     int8           `json:"gender"`
    Age        *int           `json:"age"`
    Phone      string         `json:"phone"`
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
    Version    *int           `json:"version"`
}

func (SysEmployee) TableName() string {
    return "sys_employee"
}
```

**要点**：
- 密码字段用 `json:"-"` 隐藏，永远不会序列化到响应中
- 枚举类型定义成命名常量，不硬编码魔法数字
- `Password` 用 `*string` 指针，nil 表示未设置

常量放在 `internal/model/constants.go` 统一管理：

```go
// internal/model/constants.go

package model

// 密码规则
const (
    PasswordMinLen = 8
    PasswordMaxLen = 64
)

// 字段长度上限
const (
    MaxLenRealName   = 50
    MaxLenDepartment = 50
    MaxLenPosition   = 50
    MaxLenEmail      = 100
    MaxLenRemark     = 200
)
```

---

### 第 2 步：DTO（数据传输对象）

DTO 分离了「API 请求结构」和「数据库模型」，避免内部数据泄露。

```go
// internal/dto/employee.go

package dto

import (
    "shopmall/internal/model"
    "time"
)

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
```

**要点**：
- 请求 DTO 只包含客户端需要传的字段
- 响应 DTO 只包含允许客户端看到的字段（绝无 Password）
- `omitempty` 标签让可选字段在 JSON 中不传时序列化为 null/省略

---

### 第 3 步：Repository（数据访问层）

Repository 是唯一直接写 SQL 的地方，使用 `database/sql` 原生操作。

```go
// internal/repository/employee.go

package repository

import (
    "context"
    "database/sql"
    "fmt"
    "shopmall/internal/model"
    "shopmall/pkg"
    "go.uber.org/zap"
)

// EmployeeRepo 员工数据访问层
type EmployeeRepo struct {
    db *sql.DB
}

// NewEmployeeRepo 创建 EmployeeRepo（Wire Provider）
func NewEmployeeRepo(db *sql.DB) *EmployeeRepo {
    return &EmployeeRepo{db: db}
}

// InsertEmployee 插入新员工记录
func (r *EmployeeRepo) InsertEmployee(ctx context.Context, emp *model.SysEmployee) error {
    const query = `INSERT INTO sys_employee
        (password, real_name, avatar, gender, age, phone, email,
         department, position, role, work_status, can_login,
         create_by, update_by, remark)
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

// GetByPhone 按手机号查询（用于去重检查）
func (r *EmployeeRepo) GetByPhone(ctx context.Context, phone string) (*model.SysEmployee, error) {
    const query = `SELECT employee_id, password, real_name, avatar, gender, age,
        phone, email, department, position, role, work_status, can_login,
        create_time, update_time, create_by, update_by, remark, deleted_at, version
        FROM sys_employee WHERE phone = ? AND deleted_at IS NULL LIMIT 1`

    row := r.db.QueryRowContext(ctx, query, phone)
    emp := &model.SysEmployee{}
    if err := row.Scan(
        &emp.EmployeeID, &emp.Password, &emp.RealName, &emp.Avatar,
        &emp.Gender, &emp.Age, &emp.Phone, &emp.Email,
        &emp.Department, &emp.Position, &emp.Role,
        &emp.WorkStatus, &emp.CanLogin, &emp.CreateTime, &emp.UpdateTime,
        &emp.CreateBy, &emp.UpdateBy, &emp.Remark, &emp.DeletedAt, &emp.Version,
    ); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        pkg.Logger.Error("[EmployeeRepo] GetByPhone failed", zap.Error(err))
        return nil, err
    }
    return emp, nil
}
```

**要点**：
- **永远用 `?` 占位符**，禁止字符串拼接 SQL（防注入）
- 用 `ExecContext` 执行写操作，`QueryRowContext` 执行单行查询
- 错误包装用 `fmt.Errorf("...: %w", err)` 保留调用链
- `LastInsertId()` 回填自增 ID 到 Model
- 软删除用 `WHERE deleted_at IS NULL`

---

### 第 4 步：Service（业务逻辑层）

Service 是 Write 操作的核心：校验、加密、组装、调用 Repo。

```go
// internal/service/employee.go

package service

import (
    "context"
    "errors"
    "fmt"
    "regexp"

    commonutil "shopmall/common/util"
    "shopmall/internal/dto"
    "shopmall/internal/model"
    "shopmall/internal/repository"
    "golang.org/x/crypto/bcrypt"
)

var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

// IsValidPhone 校验手机号格式
func IsValidPhone(phone string) bool {
    return phoneRegex.MatchString(phone)
}

// HashPassword 生成 bcrypt 密码哈希（cost=12，符合 OWASP 建议）
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    return string(bytes), err
}

// MaskPhone 手机号脱敏：保留前3位和后4位
func MaskPhone(phone string) string {
    n := len(phone)
    if n <= 7 {
        return phone
    }
    return phone[:3] + "****" + phone[n-4:]
}

// CreateEmployee 创建员工（核心 Write 方法）
func (s *EmployeeService) CreateEmployee(ctx context.Context, req *dto.CreateEmployeeRequest) (*dto.EmployeeInfo, error) {
    // 1. 必填校验
    if req.RealName == "" {
        return nil, errors.New("姓名不能为空")
    }
    if req.Phone == "" {
        return nil, errors.New("手机号不能为空")
    }
    if !IsValidPhone(req.Phone) {
        return nil, errors.New("手机号格式不正确")
    }

    // 2. 唯一性校验
    if existing, err := s.employeeRepo.GetByPhone(ctx, req.Phone); err == nil && existing != nil {
        return nil, fmt.Errorf("手机号 %s 已被使用", req.Phone)
    }

    // 3. 枚举值校验（设置默认值）
    if req.Gender == 0 {
        req.Gender = int8(model.GenderUnknown)
    }
    if req.WorkStatus == 0 {
        req.WorkStatus = int8(model.WorkStatusOnJob)
    }
    if req.CanLogin == 0 {
        req.CanLogin = int8(model.CanLoginYes)
    }

    // 4. 密码处理（复杂度校验 + bcrypt 加密）
    if req.Password == "" {
        return nil, errors.New("创建员工必须设置密码")
    }
    if len(req.Password) < model.PasswordMinLen || len(req.Password) > model.PasswordMaxLen {
        return nil, errors.New("密码长度必须在8-64位之间")
    }
    if !checkPasswordComplexity(req.Password) {
        return nil, errors.New("密码必须包含大写字母、小写字母、数字和特殊字符")
    }
    passwordHash, err := HashPassword(req.Password)
    if err != nil {
        return nil, errors.New("密码加密失败")
    }

    // 5. 组装 Model
    var avatar *string
    if req.Avatar != "" {
        avatar = &req.Avatar
    }

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
        Remark:     req.Remark,
    }

    // 6. 调用 Repository 执行 INSERT
    if err := s.employeeRepo.InsertEmployee(ctx, emp); err != nil {
        return nil, err
    }

    // 7. 组装返回 DTO（脱敏处理）
    return &dto.EmployeeInfo{
        ID:         emp.EmployeeID,
        RealName:   emp.RealName,
        Role:       emp.Role,
        Phone:      MaskPhone(emp.Phone),   // 脱敏
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

// checkPasswordComplexity 校验密码复杂度（大写+小写+数字+特殊字符）
func checkPasswordComplexity(password string) bool {
    var hasUpper, hasLower, hasDigit, hasSpecial bool
    for _, c := range password {
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
    return hasUpper && hasLower && hasDigit && hasSpecial
}
```

**Write 操作的标准流程**：

```
1. 参数校验（必填、格式、长度）
2. 唯一性校验（查数据库）
3. 枚举默认值处理
4. 密码加密 / 数据转换
5. 组装 Model
6. 调用 Repository 写入
7. 组装返回 DTO（脱敏）
```

**要点**：
- Service 不关心 HTTP 状态码，只返回 `data, error`
- 密码明文只在 Service 中存在，进入 Repo 时已经是 hash
- 敏感信息（手机号、头像 URL）在返回前脱敏

---

### 第 5 步：Handler（控制器层）

Handler 负责 HTTP 协议相关的事情：解析请求、调用 Service、返回 JSON。

```go
// internal/handler/employee.go

package handler

import (
    "context"
    "errors"
    "net/http"
    "strings"
    "time"

    "shopmall/common"
    "shopmall/internal/dto"
    "shopmall/internal/model"
    "shopmall/internal/service"
    "shopmall/pkg"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// EmployeeHandler 员工控制器
type EmployeeHandler struct {
    EmployeeService *service.EmployeeService
    // ... 其他依赖
}

// NewEmployeeHandler 创建 Handler（Wire Provider）
func NewEmployeeHandler(svc *service.EmployeeService, ...) *EmployeeHandler {
    return &EmployeeHandler{EmployeeService: svc}
}

// Create 添加员工
// POST /api/employee
func (h *EmployeeHandler) Create(c *gin.Context) {
    // 1. 设置请求超时上下文
    ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
    defer cancel()

    // 2. 解析请求体
    var req dto.CreateEmployeeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, "请求参数错误"))
        return
    }

    // 3. Handler 层的字段长度校验（快速拦截）
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

    // 4. 调用 Service 执行业务逻辑
    emp, err := h.EmployeeService.CreateEmployee(ctx, &req)
    if err != nil {
        // 按错误类型返回不同 HTTP 状态码
        if strings.Contains(err.Error(), "已被使用") {
            common.WriteJSON(c, http.StatusConflict, common.Fail(409, "该手机号已被使用"))
            return
        }
        if errors.Is(err, service.ErrInvalidParam) {
            common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, err.Error()))
            return
        }
        // 未知错误：记录日志，返回 500
        pkg.Logger.Error("CreateEmployee failed", zap.Error(err))
        common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "创建失败"))
        return
    }

    // 5. 成功：201 Created
    common.WriteJSON(c, http.StatusCreated, common.OkData(emp))
}
```

**要点**：
- `context.WithTimeout` 限制请求最长处理时间，防止慢查询拖垮服务
- Handler 做快速校验（格式、长度），Service 做业务校验（唯一性、复杂度）
- 错误按类型区分：参数错误 400、冲突 409、未知错误 500
- 写操作成功返回 `201 Created`，读操作返回 `200 OK`

---

### 第 6 步：Route（路由注册）

```go
// routes/routes.go

func RegisterRoutes(r *gin.Engine, hc *HealthChecker, empHandler *handler.EmployeeHandler) {
    // 健康检查
    r.GET("/health", healthCheckHandler(hc))

    // API 路由组
    api := r.Group("/api")
    {
        emp := api.Group("/employee")
        {
            emp.POST("/login", empHandler.Login)
            emp.GET("/list", empHandler.List)
            emp.POST("", empHandler.Create)          // 添加员工
            emp.PUT("/:id", empHandler.Update)       // 更新员工
            emp.PUT("/:id/password", empHandler.ResetPassword)
        }

        auth := api.Group("/auth")
        {
            auth.POST("/refresh", empHandler.RefreshAccessToken)
            auth.POST("/logout", empHandler.Logout)
        }

        user := api.Group("/user")
        {
            user.GET("/me", empHandler.GetCurrentUser)
            user.PUT("/password", empHandler.ChangePassword)
        }
    }
}
```

**API 清单**：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/employee/login` | 登录 |
| POST | `/api/employee` | **添加员工** |
| GET | `/api/employee/list` | 员工列表 |
| PUT | `/api/employee/:id` | 更新员工 |
| PUT | `/api/employee/:id/password` | 重置密码 |
| POST | `/api/auth/refresh` | 刷新 Token |
| POST | `/api/auth/logout` | 登出 |
| GET | `/api/user/me` | 当前用户信息 |
| PUT | `/api/user/password` | 修改密码 |

---

### 第 7 步：Wire（依赖注入串联）

在 `wire.go` 中注册所有 Provider，Wire 自动推导调用顺序：

```go
// cmd/server/wire.go

var wireSet = wire.NewSet(
    // 基础设施
    db.InitMySQL,           // *config.DBConfig → *sql.DB
    cache.InitRedis,        // *config.RedisConfig → *redis.Client

    // Repository 层
    repository.NewEmployeeRepo,  // *sql.DB → *repository.EmployeeRepo

    // Service 层
    service.NewEmployeeSvc,      // *repository.EmployeeRepo, *redis.Client → *service.EmployeeService
    service.NewRoleSvc,          // (无依赖) → *service.RoleService

    // Middleware
    middleware.NewLoginRateLimiter,  // → *middleware.LoginRateLimiter

    // Handler 层
    handler.NewEmployeeHandler,  // *service.EmployeeService, *service.RoleService, ... → *handler.EmployeeHandler

    // App 组装
    newApp,  // 收集所有依赖
)
```

Wire 自动生成的调用链：

```
*config.Config (main.go 传入)
    │
    ├── .DB ──→ InitMySQL() ──→ *sql.DB
    │                                 │
    ├── .Redis ──→ InitRedis() ──→ *redis.Client
    │                                 │
    └─────────────────────────────────┼──────────────────────────────────┐
                                      ▼                                  ▼
                            NewEmployeeRepo(db)                NewEmployeeSvc(repo, redis)
                                │                                    │
                                ▼                                    ▼
                       *repository.EmployeeRepo              *service.EmployeeService
                                                                     │
    NewRoleSvc() ──→ *service.RoleService ─────────────────────────────┤
                                                                     │
    NewLoginRateLimiter() ──→ *middleware.LoginRateLimiter ───────────┤
                                                                     │�
    NewEmployeeHandler(svc, roleSvc, limiter, redis) ─────────────────┘
                            │
                            ▼
                     *handler.EmployeeHandler
                            │
    newApp(db, redis, cfg, empHandler) ──→ *App
```

运行 `wire` 生成代码后，`main.go` 只需一行调用：

```go
app, err := server.InitializeApp(cfg)
```

所有依赖自动组装完毕，`app.EmployeeHandler` 直接可用。

---

### 完整的 Write 数据流

以 `POST /api/employee`（添加员工）为例，数据从请求到落库的完整流转：

```
1. Gin Router         →  匹配 POST /api/employee → EmployeeHandler.Create
2. Handler.Create     →  c.ShouldBindJSON(&req)  解析请求体
3. Handler.Create     →  字段长度/格式快速校验
4. Service.CreateEmployee →  必填/枚举值校验
5. Service.CreateEmployee →  employeeRepo.GetByPhone()  查手机号是否重复
6. Service.CreateEmployee →  checkPasswordComplexity()  密码复杂度校验
7. Service.CreateEmployee →  HashPassword(req.Password)  bcrypt 加密
8. Service.CreateEmployee →  组装 model.SysEmployee
9. Repository.InsertEmployee →  db.ExecContext(INSERT INTO ...)  写入 MySQL
10. Repository.InsertEmployee →  result.LastInsertId()  回填 ID
11. Service.CreateEmployee →  组装 dto.EmployeeInfo（手机号脱敏）
12. Handler.Create     →  common.WriteJSON(c, 201, OkData(emp))  返回 JSON
```

### 常见错误处理模式

```go
// Handler 中的错误分类处理
emp, err := h.EmployeeService.CreateEmployee(ctx, &req)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "已被使用"):
        // 唯一约束冲突 → 409
        common.WriteJSON(c, http.StatusConflict, common.Fail(409, "该手机号已被使用"))
    case errors.Is(err, service.ErrInvalidParam):
        // 参数不合法 → 400
        common.WriteJSON(c, http.StatusBadRequest, common.Fail(400, err.Error()))
    default:
        // 未知错误 → 记录日志 + 500
        pkg.Logger.Error("CreateEmployee failed", zap.Error(err))
        common.WriteJSON(c, http.StatusInternalServerError, common.Fail(500, "创建失败"))
    }
    return
}
```

### 新模块开发 Checklist

开发新业务模块时，按此清单逐项完成：

- [ ] 在 `internal/model/` 定义数据模型和枚举常量
- [ ] 在 `internal/model/constants.go` 添加字段长度常量
- [ ] 在 `internal/dto/` 定义请求/响应 DTO
- [ ] 在 `internal/repository/` 实现 CRUD SQL 方法
- [ ] 在 `internal/service/` 实现业务逻辑（校验 + 加密 + 调用 Repo）
- [ ] 在 `internal/handler/` 实现 HTTP Handler
- [ ] 在 `routes/routes.go` 注册路由
- [ ] 在 `cmd/server/wire.go` 添加 Provider 到 wireSet
- [ ] 运行 `wire` 重新生成 `wire_gen.go`
- [ ] `go build ./...` 确认编译通过
