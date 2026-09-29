package dto

// CreateTenantPayload bootstraps a tenant and its primary organisation.
type CreateTenantPayload struct {
	LegalEntityName string          `json:"legal_entity_name"`
	FacilityName    string          `json:"facility_name"`
	HospitalType    string          `json:"hospital_type"`
	FacilityAddress FacilityAddress `json:"facility_address"`
}

type FacilityAddress struct {
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	State    string `json:"state"`
}

// CreateTenantResult is returned after a successful tenant + primary organisation create.
type CreateTenantResult struct {
	TenantID       string `json:"tenant_id"`
	OrganisationID string `json:"organisation_id"`
}

// TenantView is the tenant section of get/update responses.
type TenantView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// OrganisationView is the organisation section of get/update responses.
type OrganisationView struct {
	ID               string          `json:"id"`
	TenantID         string          `json:"tenant_id"`
	IsPrimary        bool            `json:"is_primary"`
	LegalEntityName  string          `json:"legal_entity_name"`
	OrganisationType string          `json:"organisation_type"`
	FacilityName     string          `json:"facility_name"`
	RegistrationNo   string          `json:"registration_no,omitempty"`
	LicenseNumber    string          `json:"license_number,omitempty"`
	GSTIN            string          `json:"gstin,omitempty"`
	Address          FacilityAddress `json:"address"`
	Status           string          `json:"status"`
}

// TenantDetailResult is returned by GetTenantByID (tenant + primary organisation).
type TenantDetailResult struct {
	Tenant       TenantView       `json:"tenant"`
	Organisation OrganisationView `json:"organisation"`
}

// UpdateTenantOrgPayload updates tenant + organisation identified by organisation_id.
type UpdateTenantOrgPayload struct {
	OrganisationID  string          `json:"organisation_id"`
	LegalEntityName string          `json:"legal_entity_name"`
	FacilityName    string          `json:"facility_name"`
	HospitalType    string          `json:"hospital_type"`
	FacilityAddress FacilityAddress `json:"facility_address"`
	RegistrationNo  string          `json:"registration_no"`
	LicenseNumber   string          `json:"license_number"`
	GSTIN           string          `json:"gstin"`
	Status          string          `json:"status"` // organisation status when set
	TenantStatus    string          `json:"tenant_status"`
}
