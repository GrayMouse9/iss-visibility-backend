package ds

// User — учётная запись. Пароль хранится только как bcrypt-хэш, отсюда varchar(100).
type User struct {
	ID          uint   `gorm:"primaryKey"`
	Login       string `gorm:"type:varchar(25);unique;not null"`
	Password    string `gorm:"type:varchar(100);not null"`
	IsModerator bool   `gorm:"type:boolean;default:false"`
}
