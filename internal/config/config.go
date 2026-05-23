package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var Cfg *Config
var once sync.Once

type Config struct {
	App   AppConfig   `yaml:"app"`
	DB    DBConfig    `yaml:"db"`
	Redis RedisConfig `yaml:"redis"`
	CORS  CORSConfig  `yaml:"cors"`
}

type AppConfig struct {
	Port      string    `yaml:"port"`
	Mode      string    `yaml:"mode"`       // dev | prod | test
	JWTSecret string    `yaml:"jwt_secret"` // JWT 签名密钥，prod 必填，至少32字符
	JWT       JWTConfig `yaml:"jwt"`
}

// JWTConfig JWT 相关配置
type JWTConfig struct {
	ExpireHours        int `yaml:"expire_hours"`         // access token 有效期（小时），默认 2
	RefreshExpireHours int `yaml:"refresh_expire_hours"` // refresh token 有效期（小时），默认 168（7天）
}

type DBConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	DBName          string `yaml:"dbname"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLife     int    `yaml:"conn_max_life"`      // seconds
	ConnMaxIdleTime int    `yaml:"conn_max_idle_time"` // seconds
	TLS             string `yaml:"tls"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	TLS      bool   `yaml:"tls"`
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// Load reads config from YAML file, uses defaults for missing fields.
// 环境变量可覆盖 YAML 值（用于 docker-compose 部署）。
// 优先级：环境变量 > config.yaml > 默认值
func Load(path string) *Config {
	once.Do(func() {
		Cfg = &Config{}

		// 1. set defaults
		setDefaults(Cfg)

		// 2. read YAML file
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[Config] warning: cannot read %s: %v, using defaults", path, err)
		} else {
			if err := yaml.Unmarshal(data, Cfg); err != nil {
				log.Fatalf("[Config] failed to parse %s: %v", path, err)
			}
		}

		// 3. YAML 空值回填默认值
		fillEmptyDefaults(Cfg)

		// 4. 环境变量覆盖（docker-compose 友好）
		applyEnvOverrides(Cfg)
	})
	return Cfg
}

func setDefaults(c *Config) {
	c.App.Port = "7000"
	c.App.Mode = "dev"
	c.App.JWTSecret = "shopmall-dev-jwt-secret-key-2026!@#$%^&*"
	c.App.JWT.ExpireHours = 2
	c.App.JWT.RefreshExpireHours = 168

	c.DB.Host = "127.0.0.1"
	c.DB.Port = 3306
	c.DB.User = "root"
	c.DB.DBName = "shop_mall"
	c.DB.MaxOpenConns = 50
	c.DB.MaxIdleConns = 25
	c.DB.ConnMaxLife = 3600
	c.DB.ConnMaxIdleTime = 600

	c.Redis.Host = "127.0.0.1"
	c.Redis.Port = 6379
	c.Redis.DB = 0

	c.CORS.AllowOrigins = []string{"http://localhost:5173"}
	c.CORS.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	c.CORS.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-XSRF-TOKEN"}
	c.CORS.AllowCredentials = true
	c.CORS.MaxAge = 86400
}

// fillEmptyDefaults yaml.Unmarshal 会用零值覆盖 setDefaults，这里补回
func fillEmptyDefaults(c *Config) {
	if c.App.JWTSecret == "" {
		c.App.JWTSecret = "shopmall-dev-jwt-secret-key-2026!@#$%^&*"
	}
	if c.App.JWT.ExpireHours <= 0 {
		c.App.JWT.ExpireHours = 2
	}
	if c.App.JWT.RefreshExpireHours <= 0 {
		c.App.JWT.RefreshExpireHours = 168
	}
}

// applyEnvOverrides 环境变量覆盖 YAML 配置（docker-compose 部署场景）
//
// 支持的环境变量：
//
//	APP_PORT              - 服务端口
//	APP_MODE              - 运行模式 (dev/prod/test)
//	APP_JWT_SECRET        - JWT 签名密钥
//	APP_JWT_EXPIRE_HOURS  - access token 有效期（小时）
//	APP_JWT_REFRESH_EXPIRE_HOURS - refresh token 有效期（小时）
//	DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME - 数据库
//	REDIS_HOST, REDIS_PORT, REDIS_PASSWORD - Redis
func applyEnvOverrides(c *Config) {
	// App
	if v := os.Getenv("APP_PORT"); v != "" {
		c.App.Port = v
	}
	if v := os.Getenv("APP_MODE"); v != "" {
		c.App.Mode = v
	}
	if v := os.Getenv("APP_JWT_SECRET"); v != "" {
		c.App.JWTSecret = v
	}
	if v := os.Getenv("APP_JWT_EXPIRE_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			c.App.JWT.ExpireHours = h
		}
	}
	if v := os.Getenv("APP_JWT_REFRESH_EXPIRE_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			c.App.JWT.RefreshExpireHours = h
		}
	}

	// DB
	if v := os.Getenv("DB_HOST"); v != "" {
		c.DB.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			c.DB.Port = p
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.DB.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.DB.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.DB.DBName = v
	}

	// Redis
	if v := os.Getenv("REDIS_HOST"); v != "" {
		c.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			c.Redis.Port = p
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	}

	// CORS
	if v := os.Getenv("CORS_ALLOW_ORIGINS"); v != "" {
		origins := strings.Split(v, ",")
		c.CORS.AllowOrigins = c.CORS.AllowOrigins[:0]
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				c.CORS.AllowOrigins = append(c.CORS.AllowOrigins, origin)
			}
		}
	}
}

func (c *Config) Validate() error {
	if c.App.Port == "" {
		return fmt.Errorf("app.port not set")
	}
	if c.DB.Host == "" || c.DB.User == "" || c.DB.DBName == "" {
		return fmt.Errorf("database config incomplete: host/user/dbname")
	}
	if c.DB.Port <= 0 || c.DB.Port > 65535 {
		return fmt.Errorf("db.port invalid: %d", c.DB.Port)
	}
	if c.DB.MaxOpenConns <= 0 || c.DB.MaxOpenConns > 10000 {
		return fmt.Errorf("db.max_open_conns must be 1-10000")
	}
	if c.DB.MaxIdleConns <= 0 || c.DB.MaxIdleConns > c.DB.MaxOpenConns {
		return fmt.Errorf("db.max_idle_conns must be 1-max_open_conns(%d)", c.DB.MaxOpenConns)
	}
	if c.DB.ConnMaxLife <= 0 || c.DB.ConnMaxLife > 86400 {
		return fmt.Errorf("db.conn_max_life must be 1-86400")
	}
	if c.App.Mode == "prod" && c.DB.Password == "" {
		return fmt.Errorf("db.password required in production mode")
	}
	if c.App.Mode == "prod" && c.Redis.Password == "" {
		return fmt.Errorf("redis.password required in production mode")
	}
	if c.App.Mode == "prod" && len(c.App.JWTSecret) < 32 {
		return fmt.Errorf("app.jwt_secret must be at least 32 characters in production mode")
	}
	if c.App.JWT.ExpireHours <= 0 {
		return fmt.Errorf("app.jwt.expire_hours must be positive")
	}
	if c.App.JWT.RefreshExpireHours <= 0 {
		return fmt.Errorf("app.jwt.refresh_expire_hours must be positive")
	}
	return nil
}
