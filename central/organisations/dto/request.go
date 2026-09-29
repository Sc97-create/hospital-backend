package dto

import "time"

type OrganisationPayload struct {
	OrganisationID   string     `json:"organisation_id"`
	TenantID         string     `json:"tenant_id"`
	IsPrimary        bool       `json:"is_primary"`
	LegalEntityName  string     `json:"legal_entity_name"`
	OrganisationType string     `json:"organisation_type"`
	FacilityName     string     `json:"facility_name"`
	RegistrationNo   string     `json:"registration_no"`
	LicenseNumber    string     `json:"license_number"`
	LicenseExpiry    *time.Time `json:"license_expiry"`
	GSTIN            string     `json:"gstin"`
	Address1         string     `json:"address1"`
	Address2         string     `json:"address2"`
	CountryID        string     `json:"country_id"`
	State            string     `json:"state"`
	City             string     `json:"city"`
	Status           string     `json:"status"`

	PatientLookup bool `json:"patient_lookup"`
	LabReports    bool `json:"lab_reports"`
}
