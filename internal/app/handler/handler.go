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

// RegisterHandler Функция, в которой мы отдельно регистрируем маршруты, чтобы не писать все в одном месте.
// Адреса разбиты на два домена и начинаются с /api.
func (h *Handler) RegisterHandler(router *gin.Engine) {
	api := router.Group("/api")

	points := api.Group("/observation_points")
	{
		points.GET("", h.GetObservationPoints)
		points.GET("/feed", h.GetFeed)
		points.GET("/draft", h.GetObservationPointDraft)
		points.POST("", h.CreateObservationPointDraft)
		points.PUT("/:id/publish", h.PublishObservationPoint)
		points.DELETE("/:id", h.DeleteObservationPoint)
		points.POST("/:id/like", h.SetObservationPointLike)
	}

	users := api.Group("/users")
	{
		users.POST("", h.RegisterUser)
		users.POST("/login", h.LoginUser)
		users.POST("/logout", h.LogoutUser)
	}
}

// errorHandler пишет ошибку в журнал сервера. В теле ответа сообщений не передаём:
// код состояния — отдельный параметр HTTP-ответа, дублировать его в JSON не нужно.
func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	logrus.Error(err.Error())
	ctx.Status(errorStatusCode)
}
