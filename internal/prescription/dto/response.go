package dto

import "time"

type CreatePrescriptionResponse struct {
	Data    Data   `json:"data"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type FindManyResponse struct {
	Data       any    `json:"data"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	TotalCount int64  `json:"total_count"`
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
	MedicineResponse []MedicineResponse `json:"medicines"`
	TotalCount       int                `json:"total_count"`
	CreatedAt        time.Time          `json:"created_at"`
}
type PrescriptionListItem struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	PrescribedBy string    `json:"prescribed_by"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}
type PrescriptionPatientResponse struct {
	DoctorName string    `json:"doctor_name"`
	IssuedAt   time.Time `json:"issued_at"`
	Medicines  any       `json:"medicines"`
	Reason     string    `json:"reason"`
}
type Response struct {
	Data    any    `json:"data"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Total   int    `json:"total"`
}
