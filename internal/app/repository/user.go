package repository

import (
	"errors"

	"gorm.io/gorm/clause"

	"iss-visibility/internal/app/ds"
)

// Логин занят: обработчик отдаёт по ней 409.
var ErrLoginTaken = errors.New("логин уже занят")

// Признак модератора не задаётся при регистрации: это системное поле.
func (r *Repository) CreateUser(login, password string) (ds.User, error) {
	user := ds.User{Login: login, Password: password}

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
