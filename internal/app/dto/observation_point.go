package dto

import (
	"time"

	"iss-visibility/internal/app/repository"
)

type ObservationPoint struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ImageURL    string     `json:"image_url"`
	VideoURL    string     `json:"video_url"`
	Latitude    *float64   `json:"latitude"`
	Longitude   *float64   `json:"longitude"`
	Likes       int        `json:"likes"`
	Liked       int        `json:"liked"`
	IsMine      int        `json:"is_mine"`
	DateCreated time.Time  `json:"date_created"`
	DateFormed  *time.Time `json:"date_formed"`
}

// IsMine — признак 0/1: создатель услуги совпадает с текущим пользователем.
func NewObservationPoint(p repository.ObservationPointWithLikes, currentUserID uint) ObservationPoint {
	isMine := 0
	if p.CreatorID == currentUserID {
		isMine = 1
	}

	return ObservationPoint{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		ImageURL:    p.ImageURL,
		VideoURL:    p.VideoURL,
		Latitude:    p.Latitude,
		Longitude:   p.Longitude,
		Likes:       p.Likes,
		Liked:       p.Liked,
		IsMine:      isMine,
		DateCreated: p.DateCreated,
		DateFormed:  p.DateFormed,
	}
}

func NewObservationPoints(points []repository.ObservationPointWithLikes, currentUserID uint) []ObservationPoint {
	out := make([]ObservationPoint, 0, len(points))
	for _, p := range points {
		out = append(out, NewObservationPoint(p, currentUserID))
	}

	return out
}
