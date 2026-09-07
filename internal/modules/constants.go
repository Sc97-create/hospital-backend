package modules

import "hospital-backend/pkg/constants"

type ModuleArr []string

var ConstModules ModuleArr = ModuleArr{
	constants.Patient,
	constants.Employee,
	constants.Medicine,
	constants.Department,
	constants.Role,
	constants.Appointment,
	constants.Report,
	constants.Prescription,
	constants.Billing,
	constants.Dashboard,
}
