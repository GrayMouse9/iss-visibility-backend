package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"iss-visibility/internal/app/config"
	"iss-visibility/internal/app/middleware"
	"iss-visibility/internal/app/redis"
	"iss-visibility/internal/app/repository"
)

type Handler struct {
	Repository *repository.Repository
	Redis      *redis.Client
	Config     *config.Config
}

func NewHandler(r *repository.Repository, rd *redis.Client, c *config.Config) *Handler {
	return &Handler{
		Repository: r,
		Redis:      rd,
		Config:     c,
	}
}

// RegisterHandler Функция, в которой мы отдельно регистрируем маршруты, чтобы не писать все в одном месте.
// Адреса разбиты на два домена и начинаются с /api.
//
// Проверка токена указана у каждого маршрута отдельно, а не у группы: группы
// здесь собраны по смыслу адреса, и внутри одной группы права разные.
func (h *Handler) RegisterHandler(router *gin.Engine) {
	// Документация лежит вне /api и без проверки токена: её читают до входа.
	// Чтобы запрос со страницы прошёл, токен вставляется в окне Authorize.
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Просмотр каталога: гостю видны те же услуги, но без его отметок.
	optional := middleware.WithOptionalAuth(h.Config, h.Redis)
	// Действия от своего имени: без годного токена обработчик не вызывается.
	authorized := middleware.WithAuthCheck(h.Config, h.Redis, false)

	api := router.Group("/api")

	points := api.Group("/observation_points")
	{
		points.GET("", optional, h.GetObservationPoints)
		points.GET("/feed", optional, h.GetFeed)
		points.GET("/draft", authorized, h.GetObservationPointDraft)
		points.POST("", authorized, h.CreateObservationPointDraft)
		points.PUT("/:id/publish", authorized, h.PublishObservationPoint)
		points.DELETE("/:id", authorized, h.DeleteObservationPoint)
		points.POST("/:id/like", authorized, h.SetObservationPointLike)
	}

	users := api.Group("/users")
	{
		// Регистрация и вход открыты по определению: токена у клиента ещё нет.
		users.POST("", h.RegisterUser)
		users.POST("/login", h.LoginUser)
		users.POST("/logout", authorized, h.LogoutUser)
	}
}

// errorHandler пишет ошибку в журнал сервера. В теле ответа сообщений не передаём:
// код состояния — отдельный параметр HTTP-ответа, дублировать его в JSON не нужно.
func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	logrus.Error(err.Error())
	ctx.Status(errorStatusCode)
}
