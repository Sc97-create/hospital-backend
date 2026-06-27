package dto

type BedGenerate struct {
	BedsPerRoom    int      `json:"beds_per_room"`
	RoomNumber     []string `json:"room_number"`
	OrganisationID string   `json:"organisation_id"`
	RoomTypeID     string   `json:"room_type_id"`
}
type BedResponse struct {
	BedNumber  []string `json:"bed_number"`
	RoomNumber string   `json:"room_number"`
}
type CreateBed struct {
	RoomTypeID     string   `json:"room_type_id"`
	Beds           []ReqBed `json:"beds"`
	OrganisationID string   `json:"organisation_id"`
}
type ReqBed struct {
	RoomID    string   `json:"room_id"`
	BedsArray []string `json:"bed_array"`
}
type RoomSummaryResponse struct {
	TotalBeds   int `json:"total_beds"`
	TotalFloors int `json:"total_floors"`
	TotalRooms  int `json:"total_rooms"`
}
