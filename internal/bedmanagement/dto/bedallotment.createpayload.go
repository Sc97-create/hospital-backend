package dto

import "time"

type BedAllotmentCreatePayload struct {
	PatientID      string    `json:"patient_id"`
	BedID          string    `json:"bed_id"`
	RoomID         string    `json:"room_id"`
	OrganisationID string    `json:"organisation_id"`
	RoomTypeID     string    `json:"room_type_id"`
	BedCharges     float64   `json:"bed_charges"`
	DischargeAt    time.Time `json:"discharge_at"`
	IsEmergency    bool      `json:"is_emergency"`
}
