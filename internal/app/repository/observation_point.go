package repository

import (
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"

	"iss-visibility/internal/app/ds"
)

// Число лайков в таблице услуг не хранится — оно считается запросом.
type PointWithLikes struct {
	ds.ObservationPoint
	Likes int
}

// LEFT JOIN оставляет в выборке и услуги, которые никто не отметил.
func (r *Repository) pointsWithLikes() *gorm.DB {
	return r.db.Model(&ds.ObservationPoint{}).
		Select("observation_points.*, COUNT(point_likes.id) AS likes").
		Joins("LEFT JOIN point_likes ON point_likes.point_id = observation_points.id").
		Group("observation_points.id")
}

// Scan, в отличие от First, не считает пустой результат ошибкой.
func one(query *gorm.DB) (PointWithLikes, error) {
	var point PointWithLikes

	if err := query.Limit(1).Scan(&point).Error; err != nil {
		return point, err
	}
	if point.ID == 0 {
		return point, ErrNotFound
	}

	return point, nil
}

func (r *Repository) PublishedPoints(minLatitude float64) ([]PointWithLikes, error) {
	// тут мы пользуемся ORM
	var points []PointWithLikes

	err := r.pointsWithLikes().
		Where("status = ? AND latitude >= ?", ds.PointPublished, minLatitude).
		Order("observation_points.id").
		Scan(&points).Error

	return points, err
}

func (r *Repository) PublishedPointByID(id uint) (PointWithLikes, error) {
	// тут мы пользуемся «курсором»
	query := `SELECT observation_points.id, name, description, status, image_url, video_url,
	                 latitude, longitude, date_created, date_formed, creator_id,
	                 COUNT(point_likes.id) AS likes
	          FROM observation_points
	          LEFT JOIN point_likes ON point_likes.point_id = observation_points.id
	          WHERE observation_points.id = $1 AND status = $2
	          GROUP BY observation_points.id`

	// Создание курсора (строковый указатель)
	row := r.db.Raw(query, id, ds.PointPublished).Row()

	point := PointWithLikes{}
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
			return PointWithLikes{}, ErrNotFound
		}
		return PointWithLikes{}, err
	}

	if dateFormed.Valid {
		point.DateFormed = &dateFormed.Time
	}

	return point, nil
}

func (r *Repository) FirstPublishedPoint() (PointWithLikes, error) {
	return one(r.pointsWithLikes().
		Where("status = ?", ds.PointPublished).
		Order("observation_points.id"))
}

func (r *Repository) NextPublishedPoint(id uint) (PointWithLikes, error) {
	point, err := one(r.pointsWithLikes().
		Where("status = ? AND observation_points.id > ?", ds.PointPublished, id).
		Order("observation_points.id"))

	if errors.Is(err, ErrNotFound) {
		return r.FirstPublishedPoint() // дошли до конца — возвращаемся к началу
	}

	return point, err
}

// Черновик у пользователя один — это держит индекс idx_one_draft_per_creator.
func (r *Repository) DraftPoint(userID uint) (PointWithLikes, error) {
	return one(r.pointsWithLikes().
		Where("status = ? AND creator_id = ?", ds.PointDraft, userID))
}

func (r *Repository) CreateDraft(userID uint, name string) error {
	point := ds.ObservationPoint{
		Name:        name,
		Status:      ds.PointDraft,
		DateCreated: time.Now(),
		CreatorID:   userID,
	}

	return r.db.Omit("Creator").Create(&point).Error
}

func (r *Repository) PublishPoint(id uint, description string, latitude, longitude float64) error {
	now := time.Now()

	result := r.db.Model(&ds.ObservationPoint{}).
		Where("id = ? AND status = ?", id, ds.PointDraft).
		Updates(map[string]any{
			"description": description,
			"latitude":    latitude,
			"longitude":   longitude,
			"status":      ds.PointPublished,
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
func (r *Repository) DeletePoint(id uint) error {
	return r.db.Exec(
		"UPDATE observation_points SET status = ? WHERE id = ? AND status <> ?",
		ds.PointDeleted, id, ds.PointDeleted,
	).Error
}
