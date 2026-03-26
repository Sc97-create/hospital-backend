package models

import "time"

type RoomType struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name           string    `json:"name" gorm:"not null"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	IsDefault      bool      `json:"is_default" gorm:"default:false"`
	BasePrice      string    `json:"base_price" gorm:"default:'500'"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
