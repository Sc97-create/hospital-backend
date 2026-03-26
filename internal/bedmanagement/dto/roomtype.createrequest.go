package dto

type RoomTypeInfo struct {
	Name           string `json:"name"`
	OrganisationID string `json:"organisation_id"`
	IsDefault      bool   `json:"is_default"`
	BasePrice      string `json:"base_price"`
}
