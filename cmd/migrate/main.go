package main

import (
	"log"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"iss-visibility/internal/app/ds"
	"iss-visibility/internal/app/dsn"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("файл .env не найден, беру переменные из окружения")
	}

	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		log.Fatalf("не удалось подключиться к базе: %v", err)
	}

	err = db.AutoMigrate(
		&ds.User{},
		&ds.ObservationPoint{},
		&ds.PointLike{},
	)
	if err != nil {
		log.Fatalf("не удалось выполнить миграцию: %v", err)
	}

	err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_one_draft_per_creator
	               ON observation_points (creator_id)
	               WHERE status = 'черновик'`).Error
	if err != nil {
		log.Fatalf("не удалось создать индекс черновика: %v", err)
	}

	log.Println("Таблицы созданы")
}
