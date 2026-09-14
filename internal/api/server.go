package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"iss-visibility/internal/app/handler"
	"iss-visibility/internal/app/repository"
)

func StartServer() {
	log.Println("Starting server")

	repo, err := repository.NewRepository()
	if err != nil {
		logrus.Error("ошибка инициализации репозитория")
	}

	h := handler.NewHandler(repo)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/feed", h.GetFeed)
	r.GET("/feed/:id", h.GetFeed)
	r.GET("/add", h.GetDraft)
	r.GET("/observation_points", h.GetObservationPoints)

	r.Run()
	log.Println("Server down")
}
