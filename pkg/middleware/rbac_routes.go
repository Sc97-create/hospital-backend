package middleware

import (
	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"strings"
)

// Route action maps classify endpoints by intent (create/update/view/delete), not HTTP method.
// Some POSTs are reads (e.g. getPatients) and must live under ViewRoutes.
//
// Unmapped authenticated routes fail closed in AuthorizeRBAC.
// Intentionally unmapped until modules exist: bed/*, admins/organisationSchedule/*,
// organisation update/get (no organisation module).

var PublicRoutes = map[string]struct{}{
	"/api/v1/authentication/login":   {},
	"/api/v1/authentication/refresh": {},
	"/api/v1/authentication/logout":  {},
	"/api/v1/payment/webhook":        {},
	"/api/v1/organisation/signupOrg": {},
}

var CreateRoutes = map[string]string{
	"/api/v1/employee/addEmployee":    modules.Employee,
	"/api/v1/employee/create":         modules.Employee,
	"/api/v1/patients/addGeneralInfo": modules.Patient,
	"/api/v1/prescription/create":     modules.Prescription,
	"/api/v1/appointment/create":      modules.Appointment,
	"/api/v1/billing/create":          modules.Billing,
	"/api/v1/medicine/addMedicine":    modules.Medicine,
	"/api/v1/supplier/createSupplier": modules.Medicine,
}

var UpdateRoutes = map[string]string{
	"/api/v1/employee/update":                                modules.Employee,
	"/api/v1/prescription/updatePrescriptions":               modules.Prescription,
	"/api/v1/prescription/updatePrescriptionItem":            modules.Prescription,
	"/api/v1/prescription/updateStatus":                      modules.Prescription,
	"/api/v1/appointment/updateStatus":                       modules.Appointment,
	"/api/v1/authentication/updatePassword":                  modules.Employee,
	"/api/v1/payment/confirm":                                modules.Billing,
	"/api/v1/license/verifylicense/:organisationID":          modules.License,
	"/api/v1/billing/invoices/:invoiceID/retry-payment-link": modules.Billing,
}

var ViewRoutes = map[string]string{
	"/api/v1/role/getRoles":                                          modules.Role,
	"/api/v1/department/getDepartments":                              modules.Department,
	"/api/v1/permission/getAll":                                      modules.Role,
	"/api/v1/patients/getPatients":                                   modules.Patient,
	"/api/v1/patients/getpatientByID/:patientID":                     modules.Patient,
	"/api/v1/employee/findbyID":                                      modules.Employee,
	"/api/v1/employee/getEmployees":                                  modules.Employee,
	"/api/v1/employee/getDoctors":                                    modules.Employee,
	"/api/v1/medicine/getMedicineByID":                               modules.Medicine,
	"/api/v1/medicine/GetMedicines":                                  modules.Medicine,
	"/api/v1/medicine/searchMedicine":                                modules.Medicine,
	"/api/v1/prescription/get":                                       modules.Prescription,
	"/api/v1/prescription/getByStatus":                               modules.Prescription,
	"/api/v1/prescription/getprescriptionbyPid":                      modules.Prescription,
	"/api/v1/prescription/getPrescriptionByAppointmentID":            modules.Prescription,
	"/api/v1/prescription/getPrescriptionByPatientID":                modules.Prescription,
	"/api/v1/prescription/getMedicineInfo/:prescription_id":          modules.Prescription,
	"/api/v1/supplier/getSupplierByID":                               modules.Medicine,
	"/api/v1/supplier/getSupplierByOrgID":                            modules.Medicine,
	"/api/v1/supplier/getTotalCount":                                 modules.Medicine,
	"/api/v1/appointment/getTimeSlots":                               modules.Appointment,
	"/api/v1/appointment/getappointmentbyOrgID":                      modules.Appointment,
	"/api/v1/appointment/getAppointmentsPreview":                     modules.Appointment,
	"/api/v1/appointment/getappointmentByPatientID":                  modules.Appointment,
	"/api/v1/billing/getInvoiceByPrescriptionID/:prescriptionID":     modules.Billing,
	"/api/v1/billing/getInvoiceByAppointmentID/:appointmentID":       modules.Billing,
	"/api/v1/billing/getBillDetailsByPrescriptionID/:prescriptionID": modules.Billing,
}

var DeleteRoutes = map[string]string{
	"/api/v1/employee/delete": modules.Employee,
}

func normalizeRoutePath(path string) string {
	if path == "" {
		return ""
	}
	return strings.TrimSuffix(path, "/")
}

// ResolveRoutePermission returns module + action for a request path.
// Matches exact map keys first, then Fiber-style :param patterns (needed because
// group middleware often sees c.Route().Path as the group prefix, so callers
// should pass c.Path()).
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
