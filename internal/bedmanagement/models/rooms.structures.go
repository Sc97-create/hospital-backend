package models

import "time"

type Room struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RoomNumber     string    `json:"room_number" gorm:"type:varchar(255);not null"`
	RoomTypeID     string    `json:"room_type_id" gorm:"type:uuid;not null"`
	Status         Status    `json:"status" gorm:"default:'available'"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	Floors         int       `json:"floors" gorm:"type:integer;not null"`
}
type RoomSummary struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RoomTypeID     string    `json:"room_type_id" gorm:"type:uuid;not null"`
	TotalRooms     int       `json:"total_rooms" gorm:"type:integer;not null"`
	TotalFloors    int       `json:"total_floors" gorm:"type:integer;not null"`
	TotalBeds      int       `json:"total_beds" gorm:"type:integer;"`
	OrganisationID string    `json:"organisation_id" gorm:"type:uuid;not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
type UpdateRoomSummaryParams struct {
	TotalRooms  int
	TotalFloors int
	TotalBeds   int
	RoomTypeID  string
}
