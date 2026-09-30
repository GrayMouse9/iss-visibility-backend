package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"iss-visibility/internal/app/repository"
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

// RegisterHandler Функция, в которой мы отдельно регистрируем маршруты, чтобы не писать все в одном месте
func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/feed", h.GetFeed)
	router.GET("/feed/:id", h.GetFeed)
	router.GET("/add", h.GetObservationPointDraft)
	router.GET("/observation_points", h.GetObservationPoints)

	router.POST("/observation_points", h.CreateObservationPointDraft)
	router.POST("/observation_points/publish", h.PublishObservationPoint)
	router.POST("/observation_points/delete", h.DeleteObservationPoint)
}

// RegisterStatic То же самое, что и с маршрутами, регистрируем статику
func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/styles", "./resources/styles")
	router.Static("/icons", "./resources/icons")
	router.Static("/fonts", "./resources/fonts")
	router.Static("/img", "./resources/img")
	router.Static("/video", "./resources/video")
}

// errorHandler для более удобного вывода ошибок
func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	logrus.Error(err.Error())
	ctx.JSON(errorStatusCode, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}
