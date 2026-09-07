package routers

import (
	"hospital-backend/internal/admins"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/authentication"
	"hospital-backend/internal/bedmanagement"
	bedcontroller "hospital-backend/internal/bedmanagement/controllers"
	"hospital-backend/internal/billing"
	"hospital-backend/internal/dashboard"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/jwt"
	"hospital-backend/internal/medicine"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/payments"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/prescription"
	"hospital-backend/internal/roles"
	"hospital-backend/pkg/middleware"

	"github.com/gofiber/fiber/v2"
)

// RBACDeps holds shared auth + permission deps for protected route groups.
type RBACDeps struct {
	JWT        *jwt.JwtService
	RolePerm   middleware.RoleAccessLoader
	RoleLookup middleware.RoleIDFinder
}

func getVersion(app *fiber.App) fiber.Router {
	api := app.Group("api")
	version := api.Group("v1")
	return version
}

func RegisterRoleRoutes(app *fiber.App, service *roles.RoleServices, rbac RBACDeps) {
	version := getVersion(app)
	roleGroup := version.Group("role")
	middleware.UseProtected(roleGroup, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	roleController := roles.NewRoleControllerInterface(service)
	roleGroup.Get("/getRoles", roleController.FindMany)
}

func RegisterBedRoute(app *fiber.App, services *bedmanagement.BedContainer, jwtService *jwt.JwtService) {
	version := getVersion(app)
	bedGroup := version.Group("bed")

	// Auth only: bed routes are unmapped until a bed module exists (RBAC would fail closed).
	bedGroup.Use(func(c *fiber.Ctx) error {
		return middleware.Authenticate(c, jwtService)
	})
	RoomTypeManagement := bedcontroller.NewRoomtypeControllerInterface(services.RoomTypeService)
	RoomManagement := bedcontroller.NewRoomControllerInterface(services.RoomServices)
	BedManagement := bedcontroller.NewBedControllerInterface(services.BedServices)
	BedAllotmentManagement := bedcontroller.NewBedAllotmentController(services.BedAllotmentService)
	bedGroup.Post("/createRoomType", RoomTypeManagement.CreateRoomTypeController)
	bedGroup.Get("/getRoomTypeData", RoomTypeManagement.GetRoomTypeData)
	bedGroup.Post("/createRoom", RoomManagement.CreateRoomController)
	bedGroup.Post("/createBed", BedManagement.CreateBedController)
	bedGroup.Post("/generateBeds", BedManagement.GenerateBeds)
	bedGroup.Get("/getAvailableBeds", BedManagement.FindAllAvailableBeds)
	bedGroup.Get("/getAvailableRooms", RoomManagement.FindAllAvailableRooms)
	bedGroup.Get("/getAvailableRoomTypes", RoomTypeManagement.FindAllRoomTypes)
	bedGroup.Post("/createBedAllotment", BedAllotmentManagement.CreateBedAllotmentController)

}

func RegisterAuthRoute(app *fiber.App, service *authentication.UserService, rbac RBACDeps) {
	version := getVersion(app)
	authGroup := version.Group("authentication")
	auth := authentication.NewAuthController(service)
	authGroup.Post("/login", auth.Login)
	authGroup.Post("/refresh", auth.Refresh)
	authGroup.Post("/logout", auth.Logout)
	authGroup.Post("/requestPasswordReset", auth.RequestPasswordReset)
	// Public — password reset outside the authenticated app (email + reset token).
	authGroup.Patch("/updatePassword", auth.UpdatePassword)
	// Authenticated — first-time password change after temp-password login (JWT, RBAC-public).
	authGroup.Patch("/updatePasswordFirstLogin",
		func(c *fiber.Ctx) error { return middleware.Authenticate(c, rbac.JWT) },
		auth.UpdatePasswordFirstLogin,
	)
}

func RegisterDepartmentRoutes(app *fiber.App, service *department.DepartmentService, rbac RBACDeps) {
	version := getVersion(app)
	departmentGrp := version.Group("department")
	middleware.UseProtected(departmentGrp, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	departmentController := department.NewDepartmentControllerInterface(service)
	departmentGrp.Get("/getDepartments", departmentController.FindMany)
}

func RegisterPermissionRoutes(app *fiber.App, service *permissions.PermService, rbac RBACDeps) {
	version := getVersion(app)
	permissionGrp := version.Group("permission")
	middleware.UseProtected(permissionGrp, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	permissionGrp.Get("/getAll", func(c *fiber.Ctx) error {
		return permissions.FindMany(c, service)
	})
}

func RegisterPatientRoutes(app *fiber.App, service *patient.PatientService, rbac RBACDeps) {
	version := getVersion(app)
	patientGroup := version.Group("patients")
	middleware.UseProtected(patientGroup, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	patientManagement := patient.NewPatientControllerInterface(service)
	patientGroup.Post("/addGeneralInfo", patientManagement.AddGeneralInfoHandler)
	patientGroup.Post("/getPatients", patientManagement.Find)
	patientGroup.Get("/getpatientByID/:patientID", patientManagement.GetPatientByID)
}

func RegisterEmployeeRoutes(app *fiber.App, service *employee.EmployeeService, rbac RBACDeps) {
	version := getVersion(app)
	employeeController := employee.NewEmployeeControllerInterface(service)

	// Authenticated but RBAC-public — used across modules (e.g. doctor lookup by id).
	// Registered on its own group so UseProtected middleware below does not apply.
	employeeSearch := version.Group("employee")
	employeeSearch.Get("/getDoctors",
		func(c *fiber.Ctx) error { return middleware.Authenticate(c, rbac.JWT) },
		employeeController.FindDoctors,
	)
	employeeSearch.Get("/findbyID",
		func(c *fiber.Ctx) error { return middleware.Authenticate(c, rbac.JWT) },
		employeeController.FindByID,
	)
	// Public — first Super Admin after signupOrg; no JWT/RBAC yet.
	version.Post("/employee/create", employeeController.CreateAdmin)

	employeeGroup := version.Group("employee")
	middleware.UseProtected(employeeGroup, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	employeeGroup.Post("/addEmployee", employeeController.Add)
	employeeGroup.Patch("/update", employeeController.UpdateUser)
	employeeGroup.Delete("/delete", employeeController.Delete)
	employeeGroup.Get("/getEmployees", employeeController.FindMany)
}

func RegisterOrganisationRoutes(app *fiber.App, service *organisation.OrganisationService) {
	version := getVersion(app)
	organisationGrp := version.Group("organisation")
	organisationController := organisation.NewIOrganisationController(service)
	// signupOrg is public; other org routes stay without RBAC until an organisation module exists.
	organisationGrp.Post("/signupOrg", organisationController.CreateOrganisation)
	organisationGrp.Patch("/updateLocation", organisationController.UpdateOrganisationLoc)
	organisationGrp.Get("/getbyid/:organisation_id", organisationController.GetByID)
	organisationGrp.Patch("/update", organisationController.Update)
}

func RegisterMedicineRoutes(app *fiber.App, service *medicine.MedicineService, rbac RBACDeps) {
	version := getVersion(app)
	medicineGrp := version.Group("medicine")
	middleware.UseProtected(medicineGrp, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	medicineController := medicine.NewMedicineController(service)
	medicineGrp.Post("/addMedicine", medicineController.AddMedicine)
	medicineGrp.Get("/getMedicineByID", medicineController.GetByIDHandler)
	medicineGrp.Get("/GetMedicines", medicineController.GetAllHandler)
	medicineGrp.Get("/searchMedicine", medicineController.SearchMedicine)
}

func RegisterPrescriptionRoutes(app *fiber.App, service *prescription.PrescriptionService, pitemService *prescription.PrescriptionItemServ, rbac RBACDeps) {
	version := getVersion(app)
	prescriptionGrp := version.Group("prescription")
	middleware.UseProtected(prescriptionGrp, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)

	prescriptionController := prescription.NewPrescriptionController(service, pitemService)
	prescriptionGrp.Post("/create", prescriptionController.CreatePrescription)
	prescriptionGrp.Post("/get", prescriptionController.FindMany)
	prescriptionGrp.Get("/getByStatus", prescriptionController.FindByStatus)
	prescriptionGrp.Patch("/updatePrescriptions", prescriptionController.AddPrescriptionItems)
	prescriptionGrp.Patch("/updatePrescriptionItem", prescriptionController.UpdatePrescriptionItem)
	prescriptionGrp.Get("/getprescriptionbyPid", prescriptionController.FindPrescriptionByID)
	prescriptionGrp.Post("/getPrescriptionByAppointmentID", prescriptionController.GetPrescriptionByAppointmentID)
	prescriptionGrp.Get("/getPrescriptionByPatientID", prescriptionController.GetPrescriptionsByPatientID)
	prescriptionGrp.Patch("/updateStatus", prescriptionController.UpdateStatus)
	prescriptionGrp.Get("/getMedicineInfo/:prescription_id", prescriptionController.FindMedicineDetInfo)
}

func RegisterSupplierRoutes(app *fiber.App, service *medicine.SupplierService, rbac RBACDeps) {
	version := getVersion(app)
	supplierGrp := version.Group("supplier")
	middleware.UseProtected(supplierGrp, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	supplierController := medicine.NewSupplierController(service)
	supplierGrp.Get("/getSupplierByID", supplierController.GetSupplierByID)
	supplierGrp.Post("/getSupplierByOrgID", supplierController.GetSupplierByOrgID)
	supplierGrp.Get("/getTotalCount", supplierController.GetTotalCount)
	supplierGrp.Post("/createSupplier", supplierController.CreateSupplier)
}

func RegisterDashboardRoutes(app *fiber.App, service *dashboard.DashboardService, rbac RBACDeps) {
	version := getVersion(app)
	dashboardGrp := version.Group("dashboard")
	middleware.UseProtected(dashboardGrp, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	controller := dashboard.NewDashboardController(service)
	dashboardGrp.Get("/getByStatus", controller.GetAppointmentsGroupedByStatus)
	dashboardGrp.Get("/getTodayAppointments", controller.GetTodayLatestAppointments)
	dashboardGrp.Get("/getTodayInvoiceSummary", controller.GetTodayCompletedInvoiceSummary)
	dashboardGrp.Get("/getEmployeeStatusCounts", controller.GetEmployeeStatusCounts)
	dashboardGrp.Get("/getTodayPrescriptions", controller.GetTodayPrescriptions)
}

func RegisterAppointments(app *fiber.App, service *appointments.AppointmentService, rbac RBACDeps) {
	version := getVersion(app)
	appointmentController := appointments.NewAppointmentController(service)

	appointmentGrp := version.Group("appointment")
	middleware.UseProtected(appointmentGrp, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	appointmentGrp.Post("/create", appointmentController.CreateAppointment)
	appointmentGrp.Get("/getTimeSlots", appointmentController.GetSlots)
	appointmentGrp.Post("/getappointmentbyOrgID", appointmentController.FindManyByOrganisationID)
	appointmentGrp.Get("/getAppointmentsPreview", appointmentController.FindAppointmentsPreview)
	appointmentGrp.Patch("/updateStatus", appointmentController.UpdateStatus)
	appointmentGrp.Post("/getappointmentByPatientID", appointmentController.GetAppointmentByPatientID)
}

func RegisterOrgSchedule(app *fiber.App, service *admins.OrganisationScheduleService) {
	version := getVersion(app)
	admingrp := version.Group("admins")
	orgSched := admingrp.Group("organisationSchedule")
	// Unmapped until a dedicated module exists.
	orgSchedController := admins.NewOrgSchedController(service)
	orgSched.Post("/create", orgSchedController.Create)

}

func RegisterBillingRoutes(app *fiber.App, service *billing.InvoiceServ, rbac RBACDeps) {
	version := getVersion(app)
	billingGrp := version.Group("billing")
	middleware.UseProtected(billingGrp, rbac.JWT, rbac.RolePerm, rbac.RoleLookup)
	billingController := billing.NewBillingController(service)
	billingGrp.Post("/create", billingController.Checkout)
	billingGrp.Get("/getInvoiceByPrescriptionID/:prescriptionID", billingController.GetInvoiceByPrescriptionID)
	billingGrp.Get("/getInvoiceByAppointmentID/:appointmentID", billingController.GetInvoiceByAppointmentID)
	billingGrp.Get("/getBillDetailsByPrescriptionID/:prescriptionID", billingController.GetBillDetailsByPrescriptionID)
	billingGrp.Post("/invoices/:invoiceID/retry-payment-link", billingController.RetryPaymentLink)
}

func RegisterPaymentRoutes(app *fiber.App, payment *payments.PaymentsService, webhook *payments.IWebhookService, rbac RBACDeps) {
	version := getVersion(app)
	paymentGrp := version.Group("payment")
	paymentController := payments.NewPaymentController(payment, webhook)
	// Public — provider callbacks have no JWT; listed in PublicRoutes.
	paymentGrp.Post("/webhook", paymentController.RazorPayWebhook)
	paymentGrp.Post("/confirm",
		func(c *fiber.Ctx) error { return middleware.Authenticate(c, rbac.JWT) },
		func(c *fiber.Ctx) error { return middleware.LoadRoleAccess(c, rbac.RolePerm, rbac.RoleLookup) },
		middleware.AuthorizeRBAC,
		paymentController.UpdatePaymentManually,
	)
}
