package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"iss-visibility/internal/app/config"
	"iss-visibility/internal/app/dsn"
	"iss-visibility/internal/app/handler"
	"iss-visibility/internal/app/repository"
	"iss-visibility/internal/pkg"
)

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

	hand := handler.NewHandler(rep)

	application := pkg.NewApp(conf, router, hand)
	application.RunApp()
}
