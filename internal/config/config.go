package config

import (
	"fmt"
	"log"
	"os"
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
	Port string `yaml:"port"`
	Mode string `yaml:"mode"` // dev | prod | test
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

// Load reads config from YAML file, uses defaults for missing fields
func Load(path string) *Config {
	once.Do(func() {
		Cfg = &Config{}

		// set defaults
		setDefaults(Cfg)

		// read YAML file
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[Config] warning: cannot read %s: %v, using defaults", path, err)
			return
		}

		if err := yaml.Unmarshal(data, Cfg); err != nil {
			log.Fatalf("[Config] failed to parse %s: %v", path, err)
		}
	})
	return Cfg
}

func setDefaults(c *Config) {
	c.App.Port = "7000"
	c.App.Mode = "dev"

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
	return nil
}
