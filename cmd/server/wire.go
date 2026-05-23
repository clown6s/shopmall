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
	EmployeeRepo    *repository.EmployeeRepo
	RoleRepo        *repository.RoleRepo
	PermRepo        *repository.PermissionRepo
	EmployeeHandler *handler.EmployeeHandler
	DeptHandler     *handler.DepartmentHandler
	ProductHandler  *handler.ProductHandler
	RoleHandler     *handler.RoleHandler
	PermHandler     *handler.PermissionHandler
}

func newApp(
	db *sql.DB,
	rdb *redis.Client,
	cfg *config.Config,
	empRepo *repository.EmployeeRepo,
	roleRepo *repository.RoleRepo,
	permRepo *repository.PermissionRepo,
	empHandler *handler.EmployeeHandler,
	deptHandler *handler.DepartmentHandler,
	prodHandler *handler.ProductHandler,
	roleHandler *handler.RoleHandler,
	permHandler *handler.PermissionHandler,
) *App {
	return &App{
		DB:              db,
		Redis:           rdb,
		Cfg:             cfg,
		EmployeeRepo:    empRepo,
		RoleRepo:        roleRepo,
		PermRepo:        permRepo,
		EmployeeHandler: empHandler,
		DeptHandler:     deptHandler,
		ProductHandler:  prodHandler,
		RoleHandler:     roleHandler,
		PermHandler:     permHandler,
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
	repository.NewRoleRepo,
	repository.NewPermissionRepo,

	// Service 层
	service.NewEmployeeSvc,
	service.NewRoleSvc,
	service.NewDepartmentSvc,
	service.NewProductSvc,
	service.NewPermissionService,

	// Middleware
	middleware.NewLoginRateLimiter,

	// Handler 层
	handler.NewEmployeeHandler,
	handler.NewDepartmentHandler,
	handler.NewProductHandler,
	handler.NewRoleHandler,
	handler.NewPermissionHandler,

	// App 组装
	newApp,
)

// InitializeApp is the Wire injector entry point.
func InitializeApp(cfg *config.Config) (*App, error) {
	wire.Build(wireSet)
	return &App{}, nil
}
