package ds

type ObservationPointLike struct {
	ID                 uint `gorm:"primaryKey"`
	UserID             uint `gorm:"not null;uniqueIndex:idx_user_observation_point"`
	ObservationPointID uint `gorm:"not null;uniqueIndex:idx_user_observation_point"`

	User             User             `gorm:"foreignKey:UserID"`
	ObservationPoint ObservationPoint `gorm:"foreignKey:ObservationPointID"`
}
