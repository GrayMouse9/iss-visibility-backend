package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"iss-visibility/internal/app/auth"
	"iss-visibility/internal/app/dto"
	"iss-visibility/internal/app/repository"
)

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// RegisterUser POST /api/users — регистрация нового пользователя.
// Идентификатор и признак модератора с клиента не принимаются: это системные поля.
func (h *Handler) RegisterUser(ctx *gin.Context) {
	var request registerRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	login := strings.TrimSpace(request.Login)
	if login == "" || request.Password == "" {
		ctx.Status(http.StatusBadRequest)
		return
	}

	user, err := h.Repository.CreateUser(login, request.Password)
	switch {
	case errors.Is(err, repository.ErrLoginTaken):
		ctx.Status(http.StatusConflict)
		return
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewUser(user))
}

// LoginUser POST /api/users/login — заглушка: аутентификация появится
// в лабораторной работе №4. Пока возвращает пользователя из функции-singleton.
func (h *Handler) LoginUser(ctx *gin.Context) {
	user, err := h.Repository.UserByID(auth.CurrentUserSingleton().ID)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewUser(user))
}

// LogoutUser POST /api/users/logout — заглушка: деавторизация появится
// в лабораторной работе №4, сессии пока нет.
func (h *Handler) LogoutUser(ctx *gin.Context) {
	ctx.Status(http.StatusNoContent)
}
