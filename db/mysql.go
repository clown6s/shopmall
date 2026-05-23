package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"shopmall/internal/config"
	"shopmall/pkg"

	_ "github.com/go-sql-driver/mysql"

	"go.uber.org/zap"
)

const maxRetries = 30
const retryInterval = 2 * time.Second

func InitMySQL(cfg *config.DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName,
	)

	switch cfg.TLS {
	case "":
		dsn += "&tls=false"
	case "skip-verify":
		dsn += "&tls=skip-verify"
	default:
		dsn += "&tls=" + cfg.TLS
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("[MySQL] connection failed: %v", err)
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLife) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)

	for i := 1; i <= maxRetries; i++ {
		if err = db.Ping(); err == nil {
			if pkg.Logger != nil {
				pkg.Logger.Info("[MySQL] connected",
					zap.String("host", cfg.Host),
					zap.Int("port", cfg.Port),
					zap.String("database", cfg.DBName))
			}
			return db, nil
		}
		log.Printf("[MySQL] ping failed (retry %d/%d): %v", i, maxRetries, err)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("[MySQL] failed after %d retries: %w", maxRetries, err)
}
