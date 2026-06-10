package dto

type NewApptmnt struct {
	PatientID       string `json:"patient_id"`
	OrganisationID  string `json:"organisation_id"`
	UserID          string `json:"user_id"`
	DoctorID        string `json:"doctor_id"`
	AppointmentDate string `json:"appointment_date"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	ReasonForVisit  string `json:"reason_for_visit"`
	Notes           string `json:"notes"`
	SeriesID        string `json:"series_id"`
}
type GetDataReq struct {
	OrganisationID string  `json:"organisation_id"`
	DoctorID       string  `json:"doctor_id"`
	Date           string  `json:"date"`
	Status         string  `json:"status"`
	VisitType      string  `json:"visit_type"`
	Limit          float64 `json:"limit"`
	PageNo         float64 `json:"page_no"`
	Dblimit        int     `json:"dblimit"`
	Dbpageno       int     `json:"dbpageno"`
}
type UpdateStatus struct {
	AppointmentID string `json:"appointment_id"`
	Status        string `json:"status"`
}
type PatientAppntment struct {
	PatientID      string  `json:"patient_id"`
	OrganisationID string  `json:"organisation_id"`
	Limit          float64 `json:"limit"`
	Pageno         float64 `json:"page_no"`
	Status         string  `json:"status"`
}
