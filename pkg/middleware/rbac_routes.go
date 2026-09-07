package middleware

import (
	"hospital-backend/internal/permissions"
	"hospital-backend/pkg/constants"
	"strings"
)

// Route action maps classify endpoints by intent (create/update/view/delete), not HTTP method.
// Some POSTs are reads (e.g. getPatients) and must live under ViewRoutes.
//
// Unmapped authenticated routes fail closed in AuthorizeRBAC.
// Intentionally unmapped until modules exist: bed/*, admins/organisationSchedule/*,
// organisation update/get (no organisation module).

var PublicRoutes = map[string]struct{}{
	"/api/v1/authentication/login":                {},
	"/api/v1/authentication/refresh":              {},
	"/api/v1/authentication/logout":               {},
	"/api/v1/authentication/updatePassword":            {},
	"/api/v1/authentication/updatePasswordFirstLogin":  {},
	"/api/v1/authentication/requestPasswordReset":      {},
	"/api/v1/employee/getDoctors":           {},
	"/api/v1/employee/findbyID":             {},
	"/api/v1/employee/create":               {},
	"/api/v1/payment/webhook":               {},
	"/api/v1/organisation/signupOrg":        {},
}

var CreateRoutes = map[string]string{
	"/api/v1/employee/addEmployee":    constants.Employee,
	"/api/v1/patients/addGeneralInfo": constants.Patient,
	"/api/v1/prescription/create":     constants.Prescription,
	"/api/v1/appointment/create":      constants.Appointment,
	"/api/v1/billing/create":          constants.Billing,
	"/api/v1/medicine/addMedicine":    constants.Medicine,
	"/api/v1/supplier/createSupplier": constants.Medicine,
}

var UpdateRoutes = map[string]string{
	"/api/v1/employee/update":                                constants.Employee,
	"/api/v1/prescription/updatePrescriptions":               constants.Prescription,
	"/api/v1/prescription/updatePrescriptionItem":            constants.Prescription,
	"/api/v1/prescription/updateStatus":                      constants.Prescription,
	"/api/v1/appointment/updateStatus":                       constants.Appointment,
	"/api/v1/payment/confirm":                                constants.Billing,
	"/api/v1/billing/invoices/:invoiceID/retry-payment-link": constants.Billing,
}

var ViewRoutes = map[string]string{
	"/api/v1/role/getRoles":                                          constants.Role,
	"/api/v1/department/getDepartments":                              constants.Department,
	"/api/v1/permission/getAll":                                      constants.Role,
	"/api/v1/patients/getPatients":                                   constants.Patient,
	"/api/v1/patients/getpatientByID/:patientID":                     constants.Patient,
	"/api/v1/employee/getEmployees":                                  constants.Employee,
	"/api/v1/medicine/getMedicineByID":                               constants.Medicine,
	"/api/v1/medicine/GetMedicines":                                  constants.Medicine,
	"/api/v1/medicine/searchMedicine":                                constants.Medicine,
	"/api/v1/prescription/get":                                       constants.Prescription,
	"/api/v1/prescription/getByStatus":                               constants.Prescription,
	"/api/v1/prescription/getprescriptionbyPid":                      constants.Prescription,
	"/api/v1/prescription/getPrescriptionByAppointmentID":            constants.Prescription,
	"/api/v1/prescription/getPrescriptionByPatientID":                constants.Prescription,
	"/api/v1/prescription/getMedicineInfo/:prescription_id":          constants.Prescription,
	"/api/v1/supplier/getSupplierByID":                               constants.Medicine,
	"/api/v1/supplier/getSupplierByOrgID":                            constants.Medicine,
	"/api/v1/supplier/getTotalCount":                                 constants.Medicine,
	"/api/v1/appointment/getTimeSlots":                               constants.Appointment,
	"/api/v1/appointment/getappointmentbyOrgID":                      constants.Appointment,
	"/api/v1/appointment/getAppointmentsPreview":                     constants.Appointment,
	"/api/v1/appointment/getappointmentByPatientID":                  constants.Appointment,
	"/api/v1/billing/getInvoiceByPrescriptionID/:prescriptionID":     constants.Billing,
	"/api/v1/billing/getInvoiceByAppointmentID/:appointmentID":       constants.Billing,
	"/api/v1/billing/getBillDetailsByPrescriptionID/:prescriptionID": constants.Billing,
	"/api/v1/dashboard/getByStatus":            constants.Dashboard,
	"/api/v1/dashboard/getTodayAppointments":   constants.Dashboard,
	"/api/v1/dashboard/getTodayInvoiceSummary":  constants.Dashboard,
	"/api/v1/dashboard/getEmployeeStatusCounts": constants.Dashboard,
	"/api/v1/dashboard/getTodayPrescriptions":   constants.Dashboard,
}

var DeleteRoutes = map[string]string{
	"/api/v1/employee/delete": constants.Employee,
}

func normalizeRoutePath(path string) string {
	if path == "" {
		return ""
	}
	return strings.TrimSuffix(path, "/")
}

// ResolveRoutePermission returns module + action for a request path.
func ResolveRoutePermission(path string) (module string, action string, ok bool) {
	path = normalizeRoutePath(path)
	if path == "" {
		return "", "", false
	}
	if module, action, ok = lookupRouteMaps(path); ok {
		return module, action, true
	}
	return lookupRouteMapsPattern(path)
}

func lookupRouteMaps(path string) (module string, action string, ok bool) {
	if module, ok = CreateRoutes[path]; ok {
		return module, permissions.Create, true
	}
	if module, ok = UpdateRoutes[path]; ok {
		return module, permissions.Update, true
	}
	if module, ok = ViewRoutes[path]; ok {
		return module, permissions.View, true
	}
	if module, ok = DeleteRoutes[path]; ok {
		return module, permissions.Delete, true
	}
	return "", "", false
}

func lookupRouteMapsPattern(path string) (module string, action string, ok bool) {
	if module, ok = matchRouteMap(CreateRoutes, path); ok {
		return module, permissions.Create, true
	}
	if module, ok = matchRouteMap(UpdateRoutes, path); ok {
		return module, permissions.Update, true
	}
	if module, ok = matchRouteMap(ViewRoutes, path); ok {
		return module, permissions.View, true
	}
	if module, ok = matchRouteMap(DeleteRoutes, path); ok {
		return module, permissions.Delete, true
	}
	return "", "", false
}

func matchRouteMap(routes map[string]string, path string) (module string, ok bool) {
	for pattern, mod := range routes {
		if routePatternMatches(pattern, path) {
			return mod, true
		}
	}
	return "", false
}

func routePatternMatches(pattern, path string) bool {
	pParts := strings.Split(strings.Trim(pattern, "/"), "/")
	aParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pParts) != len(aParts) {
		return false
	}
	for i := range pParts {
		if strings.HasPrefix(pParts[i], ":") {
			continue
		}
		if pParts[i] != aParts[i] {
			return false
		}
	}
	return true
}

func IsPublicRoute(path string) bool {
	path = normalizeRoutePath(path)
	if _, ok := PublicRoutes[path]; ok {
		return true
	}
	for pattern := range PublicRoutes {
		if routePatternMatches(pattern, path) {
			return true
		}
	}
	return false
}
