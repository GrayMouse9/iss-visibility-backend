package repository

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm/clause"

	"iss-visibility/internal/app/ds"
)

// Логин занят: обработчик отдаёт по ней 409.
var ErrLoginTaken = errors.New("логин уже занят")

// Логин не найден или пароль не совпал: обработчик отдаёт по ней 401.
// Одна ошибка на оба случая намеренно: иначе по разным ответам можно
// перебором узнать, какие логины в системе существуют.
var ErrInvalidCredentials = errors.New("неверный логин или пароль")

// Признак модератора не задаётся при регистрации: это системное поле.
// Пароль в базу уходит только в виде bcrypt-хэша.
func (r *Repository) CreateUser(login, password string) (ds.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ds.User{}, err
	}

	user := ds.User{Login: login, Password: string(hash)}

	result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&user)
	if result.Error != nil {
		return ds.User{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ds.User{}, ErrLoginTaken
	}

	return user, nil
}

func (r *Repository) UserByID(id uint) (ds.User, error) {
	var user ds.User

	err := r.db.Where("id = ?", id).First(&user).Error

	return user, err
}

// UserByCredentials ищет пользователя по логину и проверяет пароль.
// Открытый пароль дальше этого метода не уходит: обработчик получает
// либо пользователя, либо ErrInvalidCredentials.
func (r *Repository) UserByCredentials(login, password string) (ds.User, error) {
	var user ds.User

	err := r.db.Where("login = ?", login).First(&user).Error
	switch {
	case errors.Is(err, ErrNotFound):
		return ds.User{}, ErrInvalidCredentials
	case err != nil:
		return ds.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return ds.User{}, ErrInvalidCredentials
	}

	return user, nil
}
