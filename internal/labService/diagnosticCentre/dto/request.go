package dto

// CreateDiagnosticCentreRequest is the body for POST /diagnostic-centres.
type CreateDiagnosticCentreRequest struct {
	OrganisationID string `json:"organisation_id"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	ContactNumber  string `json:"contact_number"`
	Email          string `json:"email"`
}

// UpdateDiagnosticCentreRequest is the body for PUT /diagnostic-centres/{id}.
type UpdateDiagnosticCentreRequest struct {
	Name          string `json:"name"`
	Address       string `json:"address"`
	ContactNumber string `json:"contact_number"`
	Email         string `json:"email"`
}

// ListDiagnosticCentresRequest is query input for GET /diagnostic-centres.
type ListDiagnosticCentresRequest struct {
	OrganisationID string  `json:"organisation_id" query:"organisation_id"`
	Search         string  `json:"search" query:"search"`
	Status         string  `json:"status" query:"status"`
	Limit          float64 `json:"limit" query:"limit"`
	PageNo         float64 `json:"page_no" query:"page_no"`
	DBLimit        int     `json:"-"`
	DBOffset       int     `json:"-"`
}

// DiagnosticCentreResponse is the API shape for a diagnostic centre.
type DiagnosticCentreResponse struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisation_id"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	ContactNumber  string `json:"contact_number"`
	Email          string `json:"email"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
