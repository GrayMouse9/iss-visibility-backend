package pkg

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"iss-visibility/internal/app/config"
	"iss-visibility/internal/app/handler"
	"iss-visibility/internal/app/redis"
)

// Application связывает настройки, маршрутизатор, обработчики и клиент Redis.
type Application struct {
	Config  *config.Config
	Router  *gin.Engine
	Handler *handler.Handler
	Redis   *redis.Client
}

func NewApp(c *config.Config, r *gin.Engine, h *handler.Handler, rd *redis.Client) *Application {
	return &Application{
		Config:  c,
		Router:  r,
		Handler: h,
		Redis:   rd,
	}
}

func (a *Application) RunApp() {
	logrus.Info("Server start up")

	a.Handler.RegisterHandler(a.Router)

	serverAddress := fmt.Sprintf("%s:%d", a.Config.ServiceHost, a.Config.ServicePort)
	if err := a.Router.Run(serverAddress); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Server down")
}
