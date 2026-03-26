package utils

type ModuleArr []string

const (
	Patient     = "patient"
	Employee    = "employee"
	Medicine    = "medicine"
	Department  = "department"
	Role        = "role"
	License     = "license"
	Appointment = "appointment"
	Report      = "report"
)

var Modules ModuleArr = ModuleArr{
	Patient,
	Employee,
	Medicine,
	Department,
	Role,
	License,
	Appointment,
	Report,
}
