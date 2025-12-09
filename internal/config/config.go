package config

import (
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
		MONGO      StoreConfig
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

	cfg.MONGO = StoreConfig{
		DSN: os.Getenv("MONGO_DSN"),
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

	if err = envconfig.Process("MONGO", &cfg.MONGO); err != nil {
		return
	}

	return
}
