package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"iss-visibility/internal/app/auth"
	"iss-visibility/internal/app/config"
	"iss-visibility/internal/app/ds"
	"iss-visibility/internal/app/redis"
)

// Ключи, под которыми проверенные данные лежат в контексте запроса.
// Это единственный источник пользователя: с клиента идентификатор не принимается.
const (
	ClaimsKey = "claims"
	TokenKey  = "token"
)

// Префикс заголовка Authorization по схеме Bearer из RFC 6750.
const bearerPrefix = "Bearer "

// Токен предъявлен, но негоден: подделан, просрочен или отозван выходом.
// Отделена от сбоев сервера, потому что отвечать на неё нужно 401, а не 500.
var errTokenInvalid = errors.New("token is invalid")

// tokenFromHeader достаёт токен из заголовка.
// Пустая строка означает, что токена нет или схема не Bearer.
func tokenFromHeader(ctx *gin.Context) string {
	header := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
}

// checkToken проверяет подпись, срок и чёрный список.
// Redis опрашивается после разбора подписи: иначе запрос
// в хранилище создавала бы любая строка в заголовке.
func checkToken(ctx *gin.Context, cfg *config.Config, rd *redis.Client, token string) (*ds.JWTClaims, error) {
	claims, err := auth.ParseToken(token, cfg.JWT.Secret)
	if err != nil {
		return nil, errTokenInvalid
	}

	banned, err := rd.CheckJWTInBlacklist(ctx.Request.Context(), token)
	if err != nil {
		return nil, err
	}
	if banned {
		return nil, errTokenInvalid
	}

	return claims, nil
}

// WithAuthCheck закрывает маршрут: без годного токена обработчик не вызывается.
// requireModerator = true ставится только на маршруты модератора.
func WithAuthCheck(cfg *config.Config, rd *redis.Client, requireModerator bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := tokenFromHeader(ctx)
		if token == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := checkToken(ctx, cfg, rd, token)
		switch {
		case errors.Is(err, errTokenInvalid):
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		case err != nil:
			logrus.Error(err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		// Личность подтверждена, остаётся право доступа.
		if requireModerator && !claims.IsModerator {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		ctx.Set(ClaimsKey, claims)
		ctx.Set(TokenKey, token)

		ctx.Next()
	}
}

// WithOptionalAuth пропускает запрос без годного токена как гостевой.
// Истёкший или отозванный токен не блокирует публичное чтение.
func WithOptionalAuth(cfg *config.Config, rd *redis.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := tokenFromHeader(ctx)
		if token == "" {
			ctx.Next()
			return
		}

		claims, err := checkToken(ctx, cfg, rd, token)
		switch {
		case errors.Is(err, errTokenInvalid):
			ctx.Next()
			return
		case err != nil:
			logrus.Error(err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		ctx.Set(ClaimsKey, claims)
		ctx.Set(TokenKey, token)

		ctx.Next()
	}
}
