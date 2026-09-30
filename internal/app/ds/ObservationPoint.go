package ds

import "time"

const (
	StatusDraft     = "черновик"
	StatusPublished = "опубликован"
	StatusDeleted   = "удален"
)

type ObservationPoint struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"type:varchar(100);not null"`
	Description string `gorm:"type:varchar(500)"`
	Status      string `gorm:"type:varchar(20);not null"`

	ImageURL string `gorm:"type:varchar(200)"`
	VideoURL string `gorm:"type:varchar(200)"`

	Latitude  float64 `gorm:"not null"`
	Longitude float64 `gorm:"not null"`

	DateCreated time.Time `gorm:"not null"`
	DateFormed  *time.Time

	CreatorID uint `gorm:"not null"`
	Creator   User `gorm:"foreignKey:CreatorID"`
}
