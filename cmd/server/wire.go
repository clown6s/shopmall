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

// App application dependency container
type App struct {
	DB              *sql.DB
	Redis           *redis.Client
	Cfg             *config.Config
	EmployeeHandler *handler.EmployeeHandler
	DeptHandler     *handler.DepartmentHandler
	ProductHandler  *handler.ProductHandler
}

func newApp(
	db *sql.DB,
	rdb *redis.Client,
	cfg *config.Config,
	empHandler *handler.EmployeeHandler,
	deptHandler *handler.DepartmentHandler,
	prodHandler *handler.ProductHandler,
) *App {
	return &App{
		DB:              db,
		Redis:           rdb,
		Cfg:             cfg,
		EmployeeHandler: empHandler,
		DeptHandler:     deptHandler,
		ProductHandler:  prodHandler,
	}
}

// wireSet defines all dependency providers.
var wireSet = wire.NewSet(
	// 基础设施
	db.InitMySQL,
	cache.InitRedis,

	// Repository 层
	repository.NewEmployeeRepo,
	repository.NewDepartmentRepo,
	repository.NewProductRepo,

	// Service 层
	service.NewEmployeeSvc,
	service.NewRoleSvc,
	service.NewDepartmentSvc,
	service.NewProductSvc,

	// Middleware
	middleware.NewLoginRateLimiter,

	// Handler 层
	handler.NewEmployeeHandler,
	handler.NewDepartmentHandler,
	handler.NewProductHandler,

	// App 组装
	newApp,
)

// InitializeApp is the Wire injector entry point.
func InitializeApp(cfg *config.Config) (*App, error) {
	wire.Build(wireSet)
	return &App{}, nil
}
