package dto

import "time"

type CreatePrescriptionResponse struct {
	Data    Data   `json:"data"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type FindManyResponse struct {
	Data       []PrescriptionListItem `json:"data"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	TotalCount int64                  `json:"total_count"`
}
type Data struct {
	ID string `json:"id"`
}

type MedicineResponse struct {
	MedicineID      string  `json:"medicine_id"`
	MedicineName    string  `json:"medicine_name"`
	Frequency       Freq    `json:"frequency"`
	Quantity        int     `json:"quantity"`
	DurationDay     float64 `json:"duration_day"`
	DurationType    string  `json:"duration_type"`
	TabletForm      string  `json:"tablet_form"`
	FoodInstruction string  `json:"food_instruction"`
	MedicineType    string  `json:"medicine_type"`
	Dosage          string  `json:"dosage"`
}
type Freq struct {
	Morning   float64 `json:"morning" gorm:"column:morning"`
	Afternoon float64 `json:"afternoon" gorm:"column:afternoon"`
	Night     float64 `json:"night" gorm:"column:night"`
}
type FindPrescriptionByIDResponse struct {
	Data    FindPrescriptionByIDResponseData `json:"data"`
	Code    string                           `json:"code"`
	Message string                           `json:"message"`
}
type FindPrescriptionByIDResponseData struct {
	MedicineResponse any       `json:"medicines"`
	TotalCount       int       `json:"total_count"`
	CreatedAt        time.Time `json:"created_at"`
}
type PrescriptionListItem struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	PrescribedBy   string    `json:"prescribed_by"`
	PatientID      string    `json:"patient_id"`
	PatientName    string    `json:"patient_name"`
	AppointmentID  string    `json:"appointment_id"`
	CreatedAt      time.Time `json:"created_at"`
	Status         string    `json:"status"`
}
type AppointmentPrescriptionResponse struct {
	PrescriptionID    string                        `json:"prescription_id"`
	IssuedAt          time.Time                     `json:"issued_at"`
	DoctorName        string                        `json:"doctor_name"`
	PrescriptionItems []AppointmentPrescriptionItem `json:"prescription_items"`
}
type AppointmentPrescriptionItem struct {
	PrescriptionItemID string  `json:"prescription_item_id"`
	PrescriptionID     string  `json:"prescription_id"`
	MedicineID         string  `json:"medicine_id"`
	MedicineName       string  `json:"medicine_name"`
	MedicineForm       string  `json:"medicine_form"`
	MedicineStrength   string  `json:"medicine_strength"`
	Frequency          Freq    `json:"frequency"`
	DurationDay        float64 `json:"duration_day"`
	DurationType       string  `json:"duration_type"`
	FoodInstruction    string  `json:"food_instruction"`
	Quantity           int     `json:"quantity"`
}
type PrescriptionPatientResponse struct {
	PrescriptionID string    `json:"prescription_id"`
	DoctorName     string    `json:"doctor_name"`
	IssuedAt       time.Time `json:"issued_at"`
	Medicines      any       `json:"medicines"`
	Reason         string    `json:"reason"`
}
type Response struct {
	Data    any    `json:"data"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Total   int    `json:"total"`
}
type PrescriptionQtyInfo struct {
	PrescriptionID       string `json:"prescription_id"`
	Quantity             int64  `json:"quantity"`
	MedicineID           string `json:"medicine_id"`
	BalanceAfterDispense int    `json:"balance_after_dispense"` // total already dispensed so far
}
