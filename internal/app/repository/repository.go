package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Обработчики сравнивают с ней ошибку и отдают 404, не зная про GORM.
var ErrNotFound = gorm.ErrRecordNotFound

type Repository struct {
	db *gorm.DB
}

func New(dsn string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}
