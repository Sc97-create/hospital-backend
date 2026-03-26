package dto

type BedInfo struct {
	BedsPerRoom    int      `json:"beds_per_room"`
	RoomNumber     []string `json:"room_number"`
	OrganisationID string   `json:"organisation_id"`
}
