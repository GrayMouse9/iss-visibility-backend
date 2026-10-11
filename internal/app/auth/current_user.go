package auth

import "sync"

// Идентификатор пользователя задан константой: авторизация появится в лабораторной работе №4.
const currentUserID = 1

type CurrentUser struct {
	ID uint
}

var (
	once    sync.Once
	current *CurrentUser
)

// CurrentUserSingleton возвращает единственный экземпляр текущего пользователя.
// Все методы API берут создателя только отсюда, с клиента он не принимается.
func CurrentUserSingleton() *CurrentUser {
	once.Do(func() {
		current = &CurrentUser{ID: currentUserID}
	})

	return current
}
