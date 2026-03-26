package dto

type RoomRequest struct {
	RoomTypeID       string `json:"room_type_id"`
	OrganisationID   string `json:"organisation_id"`
	Floor            int    `json:"no_of_floor"`
	RoomPerFloor     int    `json:"room_per_floor"`
	StartingPerFloor int    `json:"starting_per_floor"`
	Prefix           string `json:"prefix"`
}
