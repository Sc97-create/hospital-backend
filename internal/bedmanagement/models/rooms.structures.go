package models

import "time"

type Room struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RoomNumber     string    `json:"room_number" gorm:"type:varchar(255);not null"`
	RoomTypeID     string    `json:"room_type_id" gorm:"type:uuid;not null"`
	Status         string    `json:"status" gorm:"default:'available'"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	Floors         int       `json:"floors" gorm:"type:integer;not null"`
}
