package dto

type RoomTypeInfo struct {
	Name           string  `json:"name"`
	OrganisationID string  `json:"organisation_id"`
	IsDefault      bool    `json:"is_default"`
	BasePrice      float64 `json:"base_price"`
}
type RoomTypeResponse struct {
	RoomTypeID   string  `json:"room_type_id"`
	RoomTypeName string  `json:"room_type_name"`
	BasePrice    float64 `json:"base_price"`
}
