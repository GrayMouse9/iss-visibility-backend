package ds

import (
	"github.com/golang-jwt/jwt/v4"
)

// JWTClaims — содержимое токена. Встроенные RegisteredClaims дают стандартные
// поля (срок, время выпуска, издатель), свои два отвечают на «кто» и «что можно».
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID      uint `json:"user_id"`
	IsModerator bool `json:"is_moderator"`
}
