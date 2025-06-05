package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type (
	Configs struct {
		APP      AppConfig
		CURRENCY ClientConfig
		MONGO    StoreConfig
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
	appTimeout, err := strconv.Atoi(os.Getenv("APP_TIMEOUT"))

	cfg.APP = AppConfig{
		Mode:    os.Getenv("APP_MODE"),
		Port:    os.Getenv("APP_PORT"),
		Path:    os.Getenv("APP_PATH"),
		Timeout: time.Duration(appTimeout),
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
