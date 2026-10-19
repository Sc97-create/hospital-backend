package dto

// CreateLabTestTemplateRequest is the body for POST /lab-test-templates/create.
type CreateLabTestTemplateRequest struct {
	OrganisationID string               `json:"organisation_id"`
	CreatedBy      string               `json:"created_by"`
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	TestCodes      []string             `json:"test_codes"`
	Tests          []LabTemplatePayload `json:"tests"`
}

// LabTemplatePayload is one test line when creating a template with full objects.
type LabTemplatePayload struct {
	TestCode    string   `json:"test_code"`
	TestName    string   `json:"test_name"`
	Specimens   []string `json:"specimens"`
	Type        string   `json:"type"`
	TATMinutes  int      `json:"tat_minutes"`
	Description string   `json:"description"`
}

// ListLabTestTemplatesRequest is query input for GET /lab-test-templates/list.
type ListLabTestTemplatesRequest struct {
	OrganisationID string  `json:"organisation_id" query:"organisation_id"`
	Status         string  `json:"status" query:"status"`
	Limit          float64 `json:"limit" query:"limit"`
	PageNo         float64 `json:"page_no" query:"page_no"`
	DBLimit        int     `json:"-"`
	DBOffset       int     `json:"-"`
}

// ListDefaultsRequest is query input for GET /lab-test-templates/defaults.
type ListDefaultsRequest struct {
	OrganisationID string `json:"organisation_id" query:"organisation_id"`
}

// DefaultsListResponse is the combined catalog + org templates payload.
type DefaultsListResponse struct {
	Defaults  []LabTemplateResponse     `json:"defaults"`
	Templates []LabTestTemplateResponse `json:"templates"`
}

// LabTemplateResponse is one catalog/test item in API responses.
type LabTemplateResponse struct {
	TestCode    string   `json:"test_code"`
	TestName    string   `json:"test_name"`
	Specimens   []string `json:"specimens"`
	Type        string   `json:"type"`
	TATMinutes  int      `json:"tat_minutes"`
	Description string   `json:"description,omitempty"`
}

// LabTestTemplateResponse is the API shape for a saved template.
type LabTestTemplateResponse struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	Tests          []LabTemplateResponse `json:"tests"`
	TestsCount     int                   `json:"tests_count"`
	Status         string                `json:"status"`
	OrganisationID string                `json:"organisation_id"`
	CreatedBy      string                `json:"created_by"`
	CreatedAt      string                `json:"created_at"`
	UpdatedAt      string                `json:"updated_at"`
}
