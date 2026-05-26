package dto

import "time"

type PatientInfo struct {
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	Age            string `json:"age"`
	Weight         string `json:"weight"`
	Gender         string `json:"gender"`
	EmailID        string `json:"email_id"`
	MobileNumber   string `json:"mobile_number"`
	BloodGroup     string `json:"blood_group"`
	Address        string `json:"address"`
	OrganisationID string `json:"organisation_id"`
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
	PatientBG      string    `json:"patient_bg"`
	PatientLVD     time.Time `json:"patient_lvd"`
	WaitingTime    string    `json:"waiting_time"`
}
type PatientListResponse struct {
	Data  []PatientResponse `json:"data"`
	Total int64             `json:"total"`
	Code  int               `json:"code"`
}
