//go:build wireinject
// +build wireinject

package server

import (
	"database/sql"

	"shopmall/cache"
	"shopmall/db"
	"shopmall/internal/config"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

// App application dependency container
type App struct {
	DB    *sql.DB
	Redis *redis.Client
	Cfg   *config.Config
}

func newApp(db *sql.DB, rdb *redis.Client, cfg *config.Config) *App {
	return &App{DB: db, Redis: rdb, Cfg: cfg}
}

// wireSet defines all dependency providers.
// Config is passed from main(), not created by Wire.
var wireSet = wire.NewSet(
	db.InitMySQL,
	cache.InitRedis,
	newApp,
)

// InitializeApp is the Wire injector entry point.
// Config is received as parameter (loaded in main.go before calling Wire).
func InitializeApp(cfg *config.Config) (*App, error) {
	wire.Build(wireSet)
	return &App{}, nil
}
