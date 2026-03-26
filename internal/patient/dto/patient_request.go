package dto

type PatientInfo struct {
	UserID          string   `json:"user_id"`
	FirstName       string   `json:"first_name"`
	LastName        string   `json:"last_name"`
	Age             string   `json:"age"`
	Weight          string   `json:"weight"`
	Gender          string   `json:"gender"`
	EmailID         string   `json:"email_id"`
	MobileNumber    string   `json:"mobile_number"`
	DoctorID        string   `json:"assign_doctor"`
	Symptoms        []string `json:"symptoms"`
	ActiveCondition string   `json:"active_condition"`
	OrganisationID  string   `json:"organisation_id"`
}
