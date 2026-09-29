package tenants

import (
	"strings"

	dto "hospital-backend/central/tenants/dto"
	wrapError "hospital-backend/shared/error"
)

var allowedHospitalTypes = map[string]struct{}{
	"hospital": {},
	"clinic":   {},
}

func validateCreatePayload(payload dto.CreateTenantPayload) error {
	required := []string{
		payload.LegalEntityName,
		payload.FacilityName,
		payload.HospitalType,
		payload.FacilityAddress.Address1,
		payload.FacilityAddress.City,
		payload.FacilityAddress.State,
	}
	for _, v := range required {
		if strings.TrimSpace(v) == "" {
			return wrapError.ErrInvalidRequest
		}
	}
	if !isAllowedHospitalType(payload.HospitalType) {
		return wrapError.ErrInvalidRequest
	}
	return nil
}

func validateUpdatePayload(payload dto.UpdateTenantOrgPayload) error {
	if strings.TrimSpace(payload.OrganisationID) == "" {
		return wrapError.ErrInvalidRequest
	}
	if typ := strings.TrimSpace(payload.HospitalType); typ != "" && !isAllowedHospitalType(typ) {
		return wrapError.ErrInvalidRequest
	}
	if !hasTenantChanges(payload) && !hasOrganisationChanges(payload) {
		return wrapError.ErrInvalidRequest
	}
	return nil
}

func hasTenantChanges(payload dto.UpdateTenantOrgPayload) bool {
	return strings.TrimSpace(payload.LegalEntityName) != "" ||
		strings.TrimSpace(payload.TenantStatus) != ""
}

func hasOrganisationChanges(payload dto.UpdateTenantOrgPayload) bool {
	return strings.TrimSpace(payload.LegalEntityName) != "" ||
		strings.TrimSpace(payload.FacilityName) != "" ||
		strings.TrimSpace(payload.HospitalType) != "" ||
		strings.TrimSpace(payload.RegistrationNo) != "" ||
		strings.TrimSpace(payload.LicenseNumber) != "" ||
		strings.TrimSpace(payload.GSTIN) != "" ||
		strings.TrimSpace(payload.Status) != "" ||
		strings.TrimSpace(payload.FacilityAddress.Address1) != "" ||
		strings.TrimSpace(payload.FacilityAddress.Address2) != "" ||
		strings.TrimSpace(payload.FacilityAddress.City) != "" ||
		strings.TrimSpace(payload.FacilityAddress.State) != ""
}

func isAllowedHospitalType(v string) bool {
	_, ok := allowedHospitalTypes[strings.ToLower(strings.TrimSpace(v))]
	return ok
}
