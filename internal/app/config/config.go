package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ServiceHost string
	ServicePort int

	JWT   JWTConfig
	Redis RedisConfig
}

type JWTConfig struct {
	Secret string
	TTL    time.Duration
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewConfig читает config.toml, а секреты — из окружения: в репозиторий они не попадают.
func NewConfig() (*Config, error) {
	var err error

	configName := "config"
	_ = godotenv.Load()
	if os.Getenv("CONFIG_NAME") != "" {
		configName = os.Getenv("CONFIG_NAME")
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.WatchConfig()

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}

	cfg.JWT.Secret = os.Getenv("JWT_SECRET")
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")

	log.Info("config parsed")

	return cfg, nil
}
