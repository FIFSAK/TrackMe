package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type (
	Configs struct {
		APP        AppConfig
		CURRENCY   ClientConfig
		CLICKHOUSE ClickhouseConfig
		POSTGRES   StoreConfig
		Redis      RedisConfig
		JWT        JWTConfig
	}

	AppConfig struct {
		Mode    string `required:"true"`
		Port    string
		Path    string
		Timeout time.Duration
	}

	ClientConfig struct {
		URL      string
		Login    string
		Password string
	}

	StoreConfig struct {
		DSN string
	}

	func (p *POSTGRES) GetDSN() string {
		// Проверяем, не переопределен ли DSN через переменную окружения
		if envDSN := os.Getenv("POSTGRES_DSN"); envDSN != "" {
			return envDSN
		}
		return p.DSN
	}

	ClickhouseConfig struct {
		ADDR     string
		UserName string
		Password string
		DB       string
	}

	RedisConfig struct {
		URL string
	}

	JWTConfig struct {
		SecretKey string `envconfig:"JWT_SECRET_KEY" default:"your-secret-key-change-in-production"`
	}
)

// New populates Configs struct with values from config file
// located at filepath and environment variables.
func New() (cfg Configs, err error) {
	root, err := os.Getwd()
	if err != nil {
		return
	}
	err = godotenv.Load(filepath.Join(root, ".env"))
	if err != nil {
		return Configs{}, err
	}

	requiredEnvVars := []string{
		"APP_MODE", "APP_PORT", "APP_PATH",
		"JWT_SECRET_KEY",
		"REDIS_URL",
		"CLICKHOUSE_ADDR", "CLICKHOUSE_USERNAME", "CLICKHOUSE_PASSWORD", "CLICKHOUSE_DB",
		"POSTGRES_DSN",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			return Configs{}, fmt.Errorf("required environment variable %s is not set", envVar)
		}
	}

	cfg.APP = AppConfig{
		Mode: os.Getenv("APP_MODE"),
		Port: os.Getenv("APP_PORT"),
		Path: os.Getenv("APP_PATH"),
	}

	cfg.JWT = JWTConfig{
		SecretKey: os.Getenv("JWT_SECRET_KEY"),
	}

	cfg.Redis = RedisConfig{
		URL: os.Getenv("REDIS_URL"),
	}

	cfg.CLICKHOUSE = ClickhouseConfig{
		ADDR:     os.Getenv("CLICKHOUSE_ADDR"),
		UserName: os.Getenv("CLICKHOUSE_USERNAME"),
		Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		DB:       os.Getenv("CLICKHOUSE_DB"),
	}

	cfg.POSTGRES = StoreConfig{
		DSN: os.Getenv("POSTGRES_DSN"),
	}

	if err = envconfig.Process("APP", &cfg.APP); err != nil {
		return
	}

	if err = envconfig.Process("CURRENCY", &cfg.CURRENCY); err != nil {
		return
	}

	return
}
