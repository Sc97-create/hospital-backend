package dto

import rpdto "hospital-backend/internal/rolepermissions/dto"

type LoginResponse struct {
	Token           string                       `json:"token"`
	RefreshToken    string                       `json:"refresh_token"`
	UserID          string                       `json:"user_id"`
	OrganisationID  string                       `json:"organisation_id"`
	RoleID          string                       `json:"role_id"`
	IsAdmin         bool                         `json:"is_admin"`
	Permissions     []rpdto.RoleModulePermission `json:"permissions"`
	Message         string                       `json:"message"`
	PasswordCleared bool                         `json:"password_cleared"`
}
