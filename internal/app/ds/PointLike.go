package ds

type PointLike struct {
	ID      uint `gorm:"primaryKey"`
	UserID  uint `gorm:"not null;uniqueIndex:idx_user_point"`
	PointID uint `gorm:"not null;uniqueIndex:idx_user_point"`

	User  User             `gorm:"foreignKey:UserID"`
	Point ObservationPoint `gorm:"foreignKey:PointID"`
}
