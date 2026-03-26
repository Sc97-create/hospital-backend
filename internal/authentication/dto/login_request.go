package dto

type LoginUser struct {
	Username string `json:"user_name"`
	Password string `json:"password"`
}
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
}
