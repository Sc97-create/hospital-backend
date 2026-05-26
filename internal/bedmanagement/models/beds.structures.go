package models

import "time"

type Status string

const (
	StatusAvailable   Status = "available"
	StatusOccupied    Status = "occupied"
	StatusMaintenance Status = "maintenance"
)

type Bed struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Beds           string    `json:"beds" gorm:"type:text;not null;uniqueIndex:idx_room_bed"`
	RoomID         string    `json:"room_id" gorm:"type:uuid;not null;uniqueIndex:idx_room_bed"`
	Status         Status    `json:"status" gorm:"default:'available'"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
}
