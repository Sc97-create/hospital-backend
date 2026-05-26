package modules

type ModuleArr []string

const (
	Patient      = "patient"
	Employee     = "employee"
	Medicine     = "medicine"
	Department   = "department"
	Role         = "role"
	License      = "license"
	Appointment  = "appointment"
	Report       = "report"
	Prescription = "prescription"
	Billing      = "billing"
)

var ConstModules ModuleArr = ModuleArr{
	Patient,
	Employee,
	Medicine,
	Department,
	Role,
	License,
	Appointment,
	Report,
	Prescription,
	Billing,
}
