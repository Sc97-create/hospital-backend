package models

import "time"

type Bed struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Beds           []string  `json:"beds" gorm:"type:text[]"`
	RoomID         string    `json:"room_id" gorm:"type:uuid;not null"`
	Status         string    `json:"status" gorm:"default:'available'"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
}
