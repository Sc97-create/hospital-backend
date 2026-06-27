package dto

type CreatePrescriptionRequest struct {
	MedicineID     string          `json:"medicine_id"`
	AppointmentID  string          `json:"appointment_id"`
	PatientID      string          `json:"patient_id"`
	PrescribedBy   string          `json:"prescribed_by"`
	MedicineArray  []MedicineArray `json:"medicine_array"`
	OrganisationID string          `json:"organisation_id"`
}
type MedicineArray struct {
	MedicineID      string  `json:"medicine_id"`
	MedicineName    string  `json:"medicine_name"`
	DurationDay     float64 `json:"duration_day"`
	DurationType    string  `json:"duration_type"`
	Quantity        int     `json:"quantity"`
	MedicineType    string  `json:"medicine_type"`
	FoodInstruction string  `json:"food_instruction"`
	Morning         float64 `json:"morning"`
	Afternoon       float64 `json:"afternoon"`
	Night           float64 `json:"night"`
	Dosage          string  `json:"dosage"`
}
type FindManyRequest struct {
	OrganisationID string `json:"organisation_id"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
}
type UpdateRequest struct {
	PrescriptionID string          `json:"prescription_id"`
	AppointmentID  string          `json:"appointment_id"`
	MedicineArr    []MedicineArray `json:"medicine_array"`
}
type PresPatients struct {
	PatientID      string  `json:"patient_id"`
	OrganisationID string  `json:"organisation_id"`
	Limit          float64 `json:"limit"`
	Pageno         float64 `json:"page_no"`
}
