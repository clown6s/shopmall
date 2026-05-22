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
│   ├── handler/handler.go       # Handler 容器
│   ├── model/                   # 数据模型（待开发）
│   ├── repository/              # 数据访问层（待开发）
│   ├── service/                 # 业务逻辑层（待开发）
│   └── dto/                     # 数据传输对象（待开发）
├── common/
│   ├── response.go              # 统一响应 R{Code, Message, Data, TraceID}
│   └── middleware/              # 中间件
│       ├── trace.go             # X-Trace-ID 请求追踪
│       ├── cors.go              # CORS 跨域
│       ├── recovery.go          # Panic 恢复
│       ├── security_headers.go  # 安全响应头（CSP/HSTS/X-Frame-Options）
│       ├── body_limit.go        # 请求体大小限制
│       └── context.go           # 请求上下文超时
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
| **Provider** | 一个普通函数，返回某个类型的实例。Wire 自动调用它来创建依赖 |
| **Provider Set** | 一组 Provider 的集合，方便复用 |
| **Injector** | 入口函数，声明"我需要什么参数，Wire 帮我组装出什么结果" |
| **wire.Build** | 在 Injector 函数里告诉 Wire 用哪些 Provider 来组装 |

### 第一步：安装 Wire

```bash
go install github.com/google/wire/cmd/wire@latest
```

### 第二步：定义 Provider 函数

Provider 就是普通 Go 函数，参数是它的依赖，返回值是它创建的实例：

```go
// internal/config/config.go

// Load 是 Config 的 Provider
// 签名：无参 → 返回 *Config
func Load(path string) *Config {
    cfg := &Config{}
    data, _ := os.ReadFile(path)
    yaml.Unmarshal(data, cfg)
    return cfg
}
```

```go
// db/mysql.go

// InitMySQL 是 *sql.DB 的 Provider
// 签名：依赖 *config.DBConfig → 返回 *sql.DB
func InitMySQL(cfg *config.DBConfig) (*sql.DB, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    if err = db.Ping(); err != nil {
        return nil, err
    }
    return db, nil
}
```

```go
// cache/redis.go

// InitRedis 是 *redis.Client 的 Provider
// 签名：依赖 *config.RedisConfig → 返回 *redis.Client
func InitRedis(cfg *config.RedisConfig) (*redis.Client, error) {
    opts := &redis.Options{
        Addr:     cfg.Addr(),
        Password: cfg.Password,
        DB:       cfg.DB,
    }
    client := redis.NewClient(opts)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, err
    }
    return client, nil
}
```

Wire 的工作原理就是**匹配函数签名**：
- `InitMySQL` 需要 `*config.DBConfig` → Wire 去找谁能提供这个类型
- Config 结构体有 `DB DBConfig` 字段 → Wire 自动提取子配置

### 第三步：定义 Provider Set（可选但推荐）

把相关的 Provider 组织到一起，方便复用：

```go
// cmd/server/wire.go

var wireSet = wire.NewSet(
    db.InitMySQL,     // 提供 *sql.DB
    cache.InitRedis,  // 提供 *redis.Client
    newApp,           // 提供 *App（组装函数）
)
```

### 第四步：编写 Injector（入口函数）

Injector 声明整个依赖图的起点和终点：

```go
// cmd/server/wire.go

//go:build wireinject
// +build wireinject

package server

import "github.com/google/wire"

type App struct {
    DB    *sql.DB
    Redis *redis.Client
    Cfg   *config.Config
}

// newApp 是一个 Provider，把所有依赖组装到 App 结构体
func newApp(db *sql.DB, rdb *redis.Client, cfg *config.Config) *App {
    return &App{DB: db, Redis: rdb, Cfg: cfg}
}

// InitializeApp 是 Injector 入口
// 告诉 Wire：
//   - 输入：*config.Config（由 main.go 传入）
//   - 输出：*App + error
func InitializeApp(cfg *config.Config) (*App, error) {
    wire.Build(wireSet)
    return &App{}, nil  // 这两行只是占位，Wire 会重写整个函数
}
```

**关键规则**：
- `//go:build wireinject` 标签让 Go 编译器忽略这个文件，运行时实际执行的是 `wire_gen.go`
- `wire.Build()` 参数列表告诉 Wire 用哪些 Provider 来组装依赖
- `return &App{}, nil` 是占位代码，不会被实际执行

### 第五步：生成代码

```bash
# 在项目根目录执行
cd cmd/server
wire
# 或
go generate ./...
```

Wire 会生成 `wire_gen.go`：

```go
// Code generated by Wire. DO NOT EDIT.
//go:generate go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server

package server

func InitializeApp(cfg *config.Config) (*App, error) {
    // Wire 自动推导的调用顺序：
    // 1. 从 cfg 中提取 DBConfig 和 RedisConfig
    // 2. 调用 InitMySQL(cfg.DB) → *sql.DB
    // 3. 调用 InitRedis(cfg.Redis) → *redis.Client
    // 4. 调用 newApp(db, redis, cfg) → *App
    dbConn, err := db.InitMySQL(&cfg.DB)
    if err != nil {
        return nil, err
    }

    redisClient, err := cache.InitRedis(&cfg.Redis)
    if err != nil {
        return nil, err
    }

    return newApp(dbConn, redisClient, cfg), nil
}
```

### 第六步：在 main.go 中使用

```go
func main() {
    cfg := config.Load("config.yaml")
    cfg.Validate()

    // 调用 Wire 生成的 Injector
    app, err := server.InitializeApp(cfg)
    if err != nil {
        log.Fatalf("init app failed: %v", err)
    }

    // app.DB, app.Redis, app.Cfg 全部就绪
    // ...
}
```

### 依赖关系图

```
*config.Config (外部传入)
    │
    ├── .DB (DBConfig) ──→ InitMySQL() ──→ *sql.DB
    │
    ├── .Redis (RedisConfig) ──→ InitRedis() ──→ *redis.Client
    │
    └── newApp(db, redis, cfg) ──→ *App
```

### 进阶：添加新业务模块

假设添加 `UserService`，需要依赖 `*sql.DB`：

```go
// 1. 定义 Provider 函数
// internal/service/user.go
func NewUserService(db *sql.DB) *UserService {
    return &UserService{db: db}
}

// 2. 添加到 Handler
// internal/handler/user.go
func NewUserHandler(svc *UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

// 3. 更新 wire.go
var wireSet = wire.NewSet(
    db.InitMySQL,
    cache.InitRedis,
    service.NewUserService,  // 新增
    handler.NewUserHandler, // 新增
    newApp,
)

// 4. 重新生成
// wire

// Wire 会自动生成：
// InitMySQL → *sql.DB → NewUserService → *UserService → NewUserHandler → *UserHandler → newApp
```

### Wire 常用 API

```go
// Provider Set：组织多个 Provider
wire.NewSet(providerA, providerB, providerC)

// 结构体字段自动注入：Wire 会自动把匹配的类型注入到结构体字段
var Set = wire.NewSet(
    wire.Struct(new(App), "DB", "Redis"), // 只注入 DB 和 Redis 字段
    // wire.Struct(new(App))               // 注入所有匹配字段
)

// 绑定接口到实现
var Set = wire.NewSet(
    wire.Bind(new(UserRepository), new(*userRepositoryImpl)),
    NewUserRepoImpl,
)

// 值 Provider：直接注入一个值
var Set = wire.NewSet(
    wire.Value("default-secret-key"), // 提供 string 类型
)

// 清理函数：依赖创建失败或关闭时调用
func InitMySQL(cfg *config.DBConfig) (*sql.DB, func(), error) {
    db, _ := sql.Open("mysql", dsn)
    cleanup := func() { db.Close() }
    return db, cleanup, nil
}
```

### 手写 vs 代码生成

本项目的 `wire_gen.go` 在初始骨架阶段是手写的（依赖链简单），后续添加业务模块后应改用 Wire 自动生成：

```bash
# 推荐方式：通过 go generate 触发
# 在任意 go 文件顶部加这行注释
//go:generate go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server

# 然后执行
go generate ./...
```

这样每次依赖变化后，`go generate` 自动重新生成 `wire_gen.go`。

### 注意事项

1. **`wire_gen.go` 不要手动编辑**，每次 `wire` 命令会完全覆盖
2. **循环依赖**：Wire 不支持循环依赖，如果出现说明架构设计有问题，需要重构
3. **同一类型多个 Provider**：使用 `wire.NewSet` 配合 `wire.Bind` 绑定接口来区分
4. **条件注入**：如果某些依赖可选，可以用 `wire.Struct` 指定字段，或拆分 Provider Set
