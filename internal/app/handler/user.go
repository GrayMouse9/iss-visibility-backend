package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"iss-visibility/internal/app/auth"
	"iss-visibility/internal/app/dto"
	"iss-visibility/internal/app/middleware"
	"iss-visibility/internal/app/repository"
)

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	ExpiresIn   int64  `json:"expires_in"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// RegisterUser POST /api/users — регистрация нового пользователя.
//
// @Summary Регистрация пользователя
// @Description Создаёт пользователя. Пароль хранится только в виде bcrypt-хэша.
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param request body handler.registerRequest true "Логин и пароль"
// @Success 201 {object} dto.User
// @Failure 400 "Пустой логин или пароль"
// @Failure 409 "Логин занят"
// @Failure 500
// @Router /users [post]
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

// LoginUser POST /api/users/login — вход и выдача токена.
//
// @Summary Вход
// @Description Проверяет пару логин-пароль и выдаёт JWT на срок из настроек сервиса.
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param request body handler.loginRequest true "Логин и пароль"
// @Success 200 {object} handler.loginResponse
// @Failure 400 "Тело запроса не разобрано"
// @Failure 401 "Неверный логин или пароль"
// @Failure 500
// @Router /users/login [post]
func (h *Handler) LoginUser(ctx *gin.Context) {
	var request loginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	user, err := h.Repository.UserByCredentials(request.Login, request.Password)
	switch {
	case errors.Is(err, repository.ErrInvalidCredentials):
		ctx.Status(http.StatusUnauthorized)
		return
	case err != nil:
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	token, err := auth.GenerateToken(user, h.Config.JWT.Secret, h.Config.JWT.TTL)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, loginResponse{
		ExpiresIn:   int64(h.Config.JWT.TTL.Seconds()),
		AccessToken: token,
		TokenType:   "Bearer",
	})
}

// LogoutUser POST /api/users/logout — деавторизация: токен уходит в чёрный список.
//
// @Summary Выход
// @Description Помещает предъявленный токен в чёрный список Redis до конца его срока действия.
// @Tags Пользователи
// @Security BearerAuth
// @Success 204 "Токен отозван"
// @Failure 401 "Токен отсутствует, негоден или уже отозван"
// @Failure 500
// @Router /users/logout [post]
func (h *Handler) LogoutUser(ctx *gin.Context) {
	claims := currentClaims(ctx)
	token := ctx.GetString(middleware.TokenKey)
	if claims == nil || claims.ExpiresAt == nil || token == "" {
		h.errorHandler(ctx, http.StatusInternalServerError, errors.New("маршрут выхода не закрыт middleware"))
		return
	}

	// Срок хранения ключа — остаток жизни токена. Дольше хранить незачем:
	// просроченный токен не пройдёт проверку подписи и без чёрного списка.
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		ctx.Status(http.StatusNoContent)
		return
	}

	if err := h.Redis.WriteJWTToBlacklist(ctx.Request.Context(), token, ttl); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
