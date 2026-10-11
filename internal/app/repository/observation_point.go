package repository

import (
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"iss-visibility/internal/app/ds"
)

// Число лайков в таблице услуг не хранится — оно считается запросом.
// Liked — признак 0/1: отметка текущего пользователя.
type ObservationPointWithLikes struct {
	ds.ObservationPoint
	Likes int
	Liked int
}

// LEFT JOIN оставляет в выборке и услуги, которые никто не отметил.
func (r *Repository) observationPointsWithLikes(userID uint) *gorm.DB {
	return r.db.Model(&ds.ObservationPoint{}).
		Select(`observation_points.*,
		        COUNT(observation_point_likes.id) AS likes,
		        COALESCE(MAX(CASE WHEN observation_point_likes.user_id = ? THEN 1 ELSE 0 END), 0) AS liked`,
			userID).
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

// Фильтрация по широте выполняется базой, а не приложением.
func (r *Repository) PublishedObservationPoints(userID uint, minLatitude float64) ([]ObservationPointWithLikes, error) {
	var points []ObservationPointWithLikes

	err := r.observationPointsWithLikes(userID).
		Where("status = ? AND latitude >= ?", ds.StatusPublished, minLatitude).
		Order("observation_points.id").
		Scan(&points).Error

	return points, err
}

func (r *Repository) PublishedObservationPointByID(userID, id uint) (ObservationPointWithLikes, error) {
	// тут мы пользуемся «курсором»
	query := `SELECT observation_points.id, name, description, status, image_url, video_url,
	                 latitude, longitude, date_created, date_formed, creator_id,
	                 COUNT(observation_point_likes.id) AS likes,
	                 COALESCE(MAX(CASE WHEN observation_point_likes.user_id = $1
	                                   THEN 1 ELSE 0 END), 0) AS liked
	          FROM observation_points
	          LEFT JOIN observation_point_likes ON observation_point_likes.observation_point_id = observation_points.id
	          WHERE observation_points.id = $2 AND status = $3
	          GROUP BY observation_points.id`

	// Создание курсора (строковый указатель)
	row := r.db.Raw(query, userID, id, ds.StatusPublished).Row()

	point := ObservationPointWithLikes{}
	var (
		latitude   sql.NullFloat64
		longitude  sql.NullFloat64
		dateFormed sql.NullTime
	)

	// Сканирование строки в структуру
	err := row.Scan(
		&point.ID,
		&point.Name,
		&point.Description,
		&point.Status,
		&point.ImageURL,
		&point.VideoURL,
		&latitude,
		&longitude,
		&point.DateCreated,
		&dateFormed,
		&point.CreatorID,
		&point.Likes,
		&point.Liked,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ObservationPointWithLikes{}, ErrNotFound
		}
		return ObservationPointWithLikes{}, err
	}

	if latitude.Valid {
		point.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		point.Longitude = &longitude.Float64
	}
	if dateFormed.Valid {
		point.DateFormed = &dateFormed.Time
	}

	return point, nil
}

func (r *Repository) FirstPublishedObservationPoint(userID uint) (ObservationPointWithLikes, error) {
	return one(r.observationPointsWithLikes(userID).
		Where("status = ?", ds.StatusPublished).
		Order("observation_points.id"))
}

func (r *Repository) NextPublishedObservationPoint(userID, id uint) (ObservationPointWithLikes, error) {
	point, err := one(r.observationPointsWithLikes(userID).
		Where("status = ? AND observation_points.id > ?", ds.StatusPublished, id).
		Order("observation_points.id"))

	if errors.Is(err, ErrNotFound) {
		return r.FirstPublishedObservationPoint(userID) // дошли до конца — возвращаемся к началу
	}

	return point, err
}

// Черновик у пользователя один — это держит индекс idx_one_draft_per_creator.
func (r *Repository) DraftObservationPoint(userID uint) (ObservationPointWithLikes, error) {
	return one(r.observationPointsWithLikes(userID).
		Where("status = ? AND creator_id = ?", ds.StatusDraft, userID))
}

// Создатель берётся из функции-singleton, с клиента он не принимается.
// Координаты и ссылки на файлы здесь не заполняются.
func (r *Repository) CreateObservationPointDraft(userID uint, name, description string) (ds.ObservationPoint, error) {
	// Индекс idx_one_draft_per_creator не даст создать второй черновик,
	// но понятную ошибку вместо нарушения ограничения вернуть лучше самим.
	_, err := r.DraftObservationPoint(userID)
	switch {
	case err == nil:
		return ds.ObservationPoint{}, ErrDraftExists
	case !errors.Is(err, ErrNotFound):
		return ds.ObservationPoint{}, err
	}

	point := ds.ObservationPoint{
		Name:        name,
		Description: description,
		Status:      ds.StatusDraft,
		DateCreated: time.Now(),
		CreatorID:   userID,
	}

	err = r.db.Omit("Creator").Create(&point).Error

	return point, err
}

// Публикация возможна только из черновика и только своего: вернуть назад в черновик нельзя.
func (r *Repository) PublishObservationPoint(id, userID uint, latitude, longitude float64) error {
	result := r.db.Model(&ds.ObservationPoint{}).
		Where("id = ? AND creator_id = ? AND status = ?", id, userID, ds.StatusDraft).
		Updates(map[string]any{
			"latitude":    latitude,
			"longitude":   longitude,
			"status":      ds.StatusPublished,
			"date_formed": time.Now(),
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
// Удалить можно только свою услугу.
func (r *Repository) DeleteObservationPoint(id, userID uint) error {
	result := r.db.Exec(
		"UPDATE observation_points SET status = ? WHERE id = ? AND creator_id = ? AND status <> ?",
		ds.StatusDeleted, id, userID, ds.StatusDeleted,
	)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Отметка «нравится» ставится и снимается одним методом: 1 — поставить, 0 — снять.
func (r *Repository) SetObservationPointLike(userID, pointID uint, value int) error {
	var status string

	err := r.db.Model(&ds.ObservationPoint{}).
		Select("status").
		Where("id = ?", pointID).
		Row().Scan(&status)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNotFound
	case err != nil:
		return err
	case status != ds.StatusPublished:
		return ErrNotFound // черновики и удалённые услуги на клиент не передаются
	}

	if value == 0 {
		return r.db.
			Where("user_id = ? AND observation_point_id = ?", userID, pointID).
			Delete(&ds.ObservationPointLike{}).Error
	}

	like := ds.ObservationPointLike{UserID: userID, ObservationPointID: pointID}

	// Уникальный индекс не даёт поставить вторую отметку: повторный запрос ничего не меняет.
	return r.db.Omit("User", "ObservationPoint").
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&like).Error
}
