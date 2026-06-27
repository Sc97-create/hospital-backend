package dto

import (
	"time"
)

type NewApptmntResp struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Code    string `json:"code"`
}
type AppointmentSlots struct {
	StartTime time.Time `json:"start_time"`
	Endtime   time.Time `json:"end_time"`
	Allow     bool      `json:"allow"`
}
type SlotResponse struct {
	Apptmnt []AppointmentSlots `json:"appointment_slots"`
	Message string             `json:"message"`
	Code    string             `json:"code"`
}
type Response struct {
	Data    any    `json:"data"`
	Total   int    `json:"total,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}
type AppointmentList struct {
	AppointmentID   string    `json:"appointment_id"`
	AppointmentCode string    `json:"appointment_code"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Next            bool      `json:"next"`
	PatientName     string    `json:"patient_name"`
	MobileNo        string    `json:"mobile_no"`
	DoctorName      string    `json:"doctor_name"`
	VisitType       string    `json:"visit_type"`
	Status          string    `json:"status"`
	AppointmentDate time.Time `json:"appointment_date"`
}
type AppointmentDetails struct {
	AppointmentID   string    `json:"appointment_id"`
	AppointmentCode string    `json:"appointment_code"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	PatientName     string    `json:"patient_name"`
	MobileNo        string    `json:"mobile_no"`
	DoctorName      string    `json:"doctor_name"`
	VisitType       string    `json:"visit_type"`
	Status          string    `json:"status"`
	AppointmentDate time.Time `json:"appointment_date"`
	Notes           string    `json:"notes"`
	Medicines       int       `json:"medicines"`
	PatientAge      int64     `json:"patient_age"`
	PatientGender   string    `json:"patient_gender"`
	DepartmentName  string    `json:"department_name"`
	SlotDuration    int64     `json:"slot_duration"`
}
type PatAppointment struct {
	AppointmentID   string    `json:"appointment_id"`
	AppointmentCode string    `json:"appointment_code"`
	AppointmentDate time.Time `json:"appointment_date"`
	StartTime       time.Time `json:"start_time"`
	Status          string    `json:"status"`
	DoctorName      string    `json:"doctor_name"`
	DepartmentName  string    `json:"department_name"`
	VisitType       string    `json:"visit_type"`
}
