package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"iss-visibility/internal/app/ds"
)

// GenerateToken собирает claims из пользователя и подписывает их секретом.
func GenerateToken(user ds.User, secret string, ttl time.Duration) (string, error) {
	claims := &ds.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "iss-visibility",
		},
		UserID:      user.ID,
		IsModerator: user.IsModerator,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

// ParseToken проверяет подпись и срок, возвращает claims.
func ParseToken(tokenStr, secret string) (*ds.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &ds.JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*ds.JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
