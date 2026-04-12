package dto

type RoomRequest struct {
	RoomTypeID       string  `json:"room_type_id"`
	OrganisationID   string  `json:"organisation_id"`
	Floor            float64 `json:"no_of_floor"`
	RoomPerFloor     float64 `json:"room_per_floor"`
	StartingPerFloor float64 `json:"starting_per_floor"`
	Prefix           string  `json:"prefix"`
}
type RoomResponse struct {
	RoomID      string `json:"room_id"`
	RoomNumbers string `json:"room_numbers"`
}
