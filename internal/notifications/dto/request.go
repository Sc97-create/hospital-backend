package dto

type Request struct {
	Recipient string

	Subject string
	Content string
}
type NotificationModel struct {
	PatientName     string `json:"patient_name"`
	AppointmentCode string `json:"appointment_code"`
	DoctorName      string `json:"doctor_name"`
	HospitalName    string `json:"hospital_name"`
	AppointmentDate string `json:"appointment_date"`
	AppointmentTime string `json:"appointment_time"`
	PatientEmail    string `json:"patient_email_id"`
	PatientID       string `json:"patient_id"`
	OrganisationID  string `json:"organisation_id"`
	PaymentLink     string `json:"payment_link"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	ExpiresAt       string `json:"expires_at"`
}
