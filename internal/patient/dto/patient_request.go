package dto

import "time"

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

type PatientResponse struct {
	PatientID      string    `json:"patient_id"`
	PatientCode    string    `json:"patient_code"`
	PatientName    string    `json:"patient_name"`
	PatientWeight  int       `json:"patient_weight"`
	PatientGender  string    `json:"patient_gender"`
	PatientPhone   string    `json:"patient_phone"`
	PatientAddress string    `json:"patient_address"`
	PatientEmail   string    `json:"patient_email"`
	PatientImage   string    `json:"patient_image"`
	PatientStatus  string    `json:"patient_status"`
	PatientAge     int       `json:"patient_age"`
	AdmissionDate  time.Time `json:"admission_date"`
}
