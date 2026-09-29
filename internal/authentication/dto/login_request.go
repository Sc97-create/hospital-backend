package dto

type LoginUser struct {
	Username string `json:"user_name"`
	Password string `json:"password"`
}

type UpdatePasswordRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type FirstLoginPasswordRequest struct {
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type RequestPasswordResetRequest struct {
	EmailID string `json:"email_id"`
}
