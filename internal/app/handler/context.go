package handler

import (
	"github.com/gin-gonic/gin"

	"iss-visibility/internal/app/ds"
	"iss-visibility/internal/app/middleware"
)

// currentClaims достаёт проверенные middleware данные токена.
// nil — обычное дело на открытых маршрутах: там токена может не быть.
func currentClaims(ctx *gin.Context) *ds.JWTClaims {
	value, ok := ctx.Get(middleware.ClaimsKey)
	if !ok {
		return nil
	}

	claims, ok := value.(*ds.JWTClaims)
	if !ok {
		return nil
	}

	return claims
}

// currentUserID возвращает идентификатор пользователя или 0 для гостя.
// Нулевого идентификатора в базе нет, поэтому гость не найдёт чужих отметок.
func currentUserID(ctx *gin.Context) uint {
	claims := currentClaims(ctx)
	if claims == nil {
		return 0
	}

	return claims.UserID
}
