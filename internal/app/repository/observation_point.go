package repository

import (
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"

	"iss-visibility/internal/app/ds"
)

// Число лайков в таблице услуг не хранится — оно считается запросом.
type ObservationPointWithLikes struct {
	ds.ObservationPoint
	Likes int
}

// LEFT JOIN оставляет в выборке и услуги, которые никто не отметил.
func (r *Repository) observationPointsWithLikes() *gorm.DB {
	return r.db.Model(&ds.ObservationPoint{}).
		Select("observation_points.*, COUNT(observation_point_likes.id) AS likes").
		Joins("LEFT JOIN observation_point_likes ON observation_point_likes.observation_point_id = observation_points.id").
		Group("observation_points.id")
}

// Scan, в отличие от First, не считает пустой результат ошибкой.
func one(query *gorm.DB) (ObservationPointWithLikes, error) {
	var point ObservationPointWithLikes

	if err := query.Limit(1).Scan(&point).Error; err != nil {
		return point, err
	}
	if point.ID == 0 {
		return point, ErrNotFound
	}

	return point, nil
}

func (r *Repository) PublishedObservationPoints(minLatitude float64) ([]ObservationPointWithLikes, error) {
	// тут мы пользуемся ORM
	var points []ObservationPointWithLikes

	err := r.observationPointsWithLikes().
		Where("status = ? AND latitude >= ?", ds.StatusPublished, minLatitude).
		Order("observation_points.id").
		Scan(&points).Error

	return points, err
}

func (r *Repository) PublishedObservationPointByID(id uint) (ObservationPointWithLikes, error) {
	// тут мы пользуемся «курсором»
	query := `SELECT observation_points.id, name, description, status, image_url, video_url,
	                 latitude, longitude, date_created, date_formed, creator_id,
	                 COUNT(observation_point_likes.id) AS likes
	          FROM observation_points
	          LEFT JOIN observation_point_likes ON observation_point_likes.observation_point_id = observation_points.id
	          WHERE observation_points.id = $1 AND status = $2
	          GROUP BY observation_points.id`

	// Создание курсора (строковый указатель)
	row := r.db.Raw(query, id, ds.StatusPublished).Row()

	point := ObservationPointWithLikes{}
	var dateFormed sql.NullTime

	// Сканирование строки в структуру
	err := row.Scan(
		&point.ID,
		&point.Name,
		&point.Description,
		&point.Status,
		&point.ImageURL,
		&point.VideoURL,
		&point.Latitude,
		&point.Longitude,
		&point.DateCreated,
		&dateFormed,
		&point.CreatorID,
		&point.Likes,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ObservationPointWithLikes{}, ErrNotFound
		}
		return ObservationPointWithLikes{}, err
	}

	if dateFormed.Valid {
		point.DateFormed = &dateFormed.Time
	}

	return point, nil
}

func (r *Repository) FirstPublishedObservationPoint() (ObservationPointWithLikes, error) {
	return one(r.observationPointsWithLikes().
		Where("status = ?", ds.StatusPublished).
		Order("observation_points.id"))
}

func (r *Repository) NextPublishedObservationPoint(id uint) (ObservationPointWithLikes, error) {
	point, err := one(r.observationPointsWithLikes().
		Where("status = ? AND observation_points.id > ?", ds.StatusPublished, id).
		Order("observation_points.id"))

	if errors.Is(err, ErrNotFound) {
		return r.FirstPublishedObservationPoint() // дошли до конца — возвращаемся к началу
	}

	return point, err
}

// Черновик у пользователя один — это держит индекс idx_one_draft_per_creator.
func (r *Repository) DraftObservationPoint(userID uint) (ObservationPointWithLikes, error) {
	return one(r.observationPointsWithLikes().
		Where("status = ? AND creator_id = ?", ds.StatusDraft, userID))
}

func (r *Repository) CreateObservationPointDraft(userID uint, name, description string) error {
	point := ds.ObservationPoint{
		Name:        name,
		Description: description,
		Status:      ds.StatusDraft,
		DateCreated: time.Now(),
		CreatorID:   userID,
	}

	return r.db.Omit("Creator").Create(&point).Error
}

func (r *Repository) PublishObservationPoint(id uint, latitude, longitude float64) error {
	now := time.Now()

	result := r.db.Model(&ds.ObservationPoint{}).
		Where("id = ? AND status = ?", id, ds.StatusDraft).
		Updates(map[string]any{
			"latitude":    latitude,
			"longitude":   longitude,
			"status":      ds.StatusPublished,
			"date_formed": now,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Логическое удаление: по заданию SQL-запросом без ORM, строка остаётся.
func (r *Repository) DeleteObservationPoint(id uint) error {
	return r.db.Exec(
		"UPDATE observation_points SET status = ? WHERE id = ? AND status <> ?",
		ds.StatusDeleted, id, ds.StatusDeleted,
	).Error
}
