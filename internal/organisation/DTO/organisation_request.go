package dto

type OrganisationPayload struct {
	OrganisationID   string `json:"organisation_id"`
	HospitalCode     string `json:"hospital_code"`
	OrganisationName string `json:"organisation_name"`
	LegalEntityName  string `json:"legal_entity_name"`
	HospitalType     string `json:"hospital_type"`
	Country          string `json:"country"`
	State            string `json:"state"`
	City             string `json:"city"`
	Timezone         string `json:"timezone"`
	AuditLogs        bool   `json:"audit_logs"`
	EmergencyAcess   bool   `json:"emergency_access"`
	FullName         string `json:"full_name"`
	EmailID          string `json:"work_email"`
	PhoneNumber      string `json:"mobile_number"`
	Password         string `json:"password"`
	ConfirmPassword  string `json:"confirm_password"`
}
