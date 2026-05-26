package models

import "time"

type BedAllotment struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PatientID    string    `json:"patient_id" gorm:"type:uuid;not null"`
	RoomID       string    `json:"room_id" gorm:"type:uuid;not null"`
	RoomTypeID   string    `json:"room_type_id" gorm:"type:uuid;not null"`
	BedID        string    `json:"bed_id" gorm:"type:uuid;not null"`
	Charges      float64   `json:"charges" gorm:"type:decimal;not null"`
	DischargedAt time.Time `json:"discharged_at" gorm:"autoUpdateTime"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	IsEmergency  bool      `json:"is_emergency" gorm:"default:false"`
}
