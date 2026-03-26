package dto

type UpdateRequest struct {
	UserID          string `json:"user_id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	EmailID         string `json:"email_id"`
	MobileNumber    string `json:"mob_no"`
}
