package dto

import "iss-visibility/internal/app/ds"

// Пароль в ответ не попадает.
type User struct {
	ID          uint   `json:"id"`
	Login       string `json:"login"`
	IsModerator bool   `json:"is_moderator"`
}

func NewUser(u ds.User) User {
	return User{
		ID:          u.ID,
		Login:       u.Login,
		IsModerator: u.IsModerator,
	}
}
