package dto

import (
	"time"

	"iss-visibility/internal/app/repository"
)

// ObservationPoint — общая часть ответа: сама услуга без признаков 0/1.
// Такой ответ отдают черновик, создание, публикация и отметка «нравится».
// Статус услуги в ответ не попадает: он живёт только в базе.
type ObservationPoint struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ImageURL    string     `json:"image_url"`
	VideoURL    string     `json:"video_url"`
	Latitude    *float64   `json:"latitude"`
	Longitude   *float64   `json:"longitude"`
	Likes       int        `json:"likes"`
	DateCreated time.Time  `json:"date_created"`
	DateFormed  *time.Time `json:"date_formed"`
}

// ObservationPointList — ответ метода списка. Встроенная структура даёт плоский
// JSON: поля услуги и признак is_mine лежат в нём на одном уровне.
type ObservationPointList struct {
	ObservationPoint
	IsMine int `json:"is_mine"`
}

// ObservationPointFeed — ответ ленты. От списка отличается только признаком:
// в ленте важно, отметил ли услугу текущий пользователь, а не кто её создал.
type ObservationPointFeed struct {
	ObservationPoint
	Liked int `json:"liked"`
}

// baseObservationPoint собирает общую часть ответа. Вынесена отдельно, чтобы
// набор полей услуги не разъезжался между методами.
func baseObservationPoint(p repository.ObservationPointWithLikes) ObservationPoint {

	return ObservationPoint{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		ImageURL:    p.ImageURL,
		VideoURL:    p.VideoURL,
		Latitude:    p.Latitude,
		Longitude:   p.Longitude,
		Likes:       p.Likes,
		DateCreated: p.DateCreated,
		DateFormed:  p.DateFormed,
	}
}

// NewObservationPoint — ответ без признаков 0/1: черновик, создание, публикация,
// отметка «нравится».
func NewObservationPoint(p repository.ObservationPointWithLikes) ObservationPoint {
	return baseObservationPoint(p)
}

// NewObservationPointList — ответ списка. Признак is_mine равен 1, когда создатель
// услуги совпадает с текущим пользователем, и 0 во всех остальных случаях.
func NewObservationPointList(p repository.ObservationPointWithLikes, currentUserID uint) ObservationPointList {
	isMine := 0
	if p.CreatorID == currentUserID {
		isMine = 1
	}

	return ObservationPointList{
		ObservationPoint: baseObservationPoint(p),
		IsMine:           isMine,
	}
}

// NewObservationPointLists — тот же ответ для всего списка сразу. Срез создаётся
// пустым, а не нулевым, чтобы при отсутствии записей в JSON ушло [], а не null.
func NewObservationPointLists(points []repository.ObservationPointWithLikes, currentUserID uint) []ObservationPointList {
	out := make([]ObservationPointList, 0, len(points))
	for _, p := range points {
		out = append(out, NewObservationPointList(p, currentUserID))
	}

	return out
}

// NewObservationPointFeed — ответ ленты. Признак liked считает база одним запросом
// вместе с количеством отметок, здесь он только переносится в ответ.
func NewObservationPointFeed(p repository.ObservationPointWithLikes) ObservationPointFeed {
	return ObservationPointFeed{
		ObservationPoint: baseObservationPoint(p),
		Liked:            p.Liked,
	}
}
