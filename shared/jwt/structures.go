package jwt

import "time"

type RefreshToken struct {
	ID           string    `json:"id" gorm:"primaryKey; type:uuid"`
	UserID       string    `json:"user_id" gorm:"uniqueIndex; not null"`
	RefreshToken string    `json:"refresh_token" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
