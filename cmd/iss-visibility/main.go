package main

import (
	"context"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	// Регистрация сгенерированной документации: пакет подключается
	// только ради функции init внутри него.
	_ "iss-visibility/docs"
	"iss-visibility/internal/app/config"
	"iss-visibility/internal/app/dsn"
	"iss-visibility/internal/app/handler"
	"iss-visibility/internal/app/redis"
	"iss-visibility/internal/app/repository"
	"iss-visibility/internal/pkg"
)

// @title ISS Visibility API
// @version 1.0
// @description Сервис определения времени видимости МКС с Земли. Предметная услуга — точка наблюдения.

// @host localhost:3001
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Токен выдаёт POST /api/users/login. Значение заголовка — "Bearer <токен>".
func main() {
	router := gin.Default()

	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	rep, errRep := repository.New(&repository.Settings{
		PostgresDSN:    dsn.FromEnv(),
		MinioEndpoint:  os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey: os.Getenv("MINIO_SECRET_KEY"),
		MinioBucket:    os.Getenv("MINIO_BUCKET"),
		MinioUseSSL:    os.Getenv("MINIO_USE_SSL") == "true",
		MinioPublicURL: os.Getenv("MINIO_PUBLIC_URL"),
	})

	if errRep != nil {
		logrus.Fatalf("error initializing repository: %v", errRep)
	}

	redisClient, errRedis := redis.New(context.Background(), conf.Redis)
	if errRedis != nil {
		logrus.Fatalf("error initializing redis: %v", errRedis)
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			logrus.Errorf("error closing redis: %v", err)
		}
	}()

	hand := handler.NewHandler(rep, redisClient, conf)

	application := pkg.NewApp(conf, router, hand, redisClient)

	application.RunApp()
}
