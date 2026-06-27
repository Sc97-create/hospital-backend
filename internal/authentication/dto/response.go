package dto

type LoginResponse struct {
	Token          string `json:"token"`
	RefreshToken   string `json:"refresh_token"`
	UserID         string `json:"user_id"`
	OrganisationID string `json:"organisation_id"`
	Message        string `json:"message"`
}
