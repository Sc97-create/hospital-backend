package customers

import (
	"strings"

	dto "hospital-backend/central/customers/dto"
	wrapError "hospital-backend/shared/error"
)

const minPasswordLen = 8

func validateSignupPayload(payload dto.SignupPayload) error {
	if strings.TrimSpace(payload.FullName) == "" {
		return wrapError.ErrInvalidRequest
	}
	email := strings.TrimSpace(payload.WorkEmail)
	if email == "" || !strings.Contains(email, "@") {
		return wrapError.ErrInvalidRequest
	}
	if len(payload.Password) < minPasswordLen {
		return wrapError.ErrInvalidRequest
	}
	return nil
}
