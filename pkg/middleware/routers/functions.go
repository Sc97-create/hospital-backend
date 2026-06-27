package routers

import (
	"hospital-backend/internal/admins"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/authentication"
	"hospital-backend/internal/bedmanagement"
	bedcontroller "hospital-backend/internal/bedmanagement/controllers"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/jwt"
	"hospital-backend/internal/license"
	"hospital-backend/internal/medicine"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/prescription"
	"hospital-backend/internal/roles"
	"hospital-backend/pkg/middleware"

	"github.com/gofiber/fiber/v2"
)

func getVersion(app *fiber.App) fiber.Router {
	api := app.Group("api")
	version := api.Group("v1")
	return version
}
func RegisterRoleRoutes(app *fiber.App, service *roles.RoleServices) {
	version := getVersion(app)
	roleGroup := version.Group("role")
	roleGroup.Get("/getRoles", func(c *fiber.Ctx) error {
		return roles.FindMany(c, service)
	})
}
func RegisterBedRoute(app *fiber.App, services *bedmanagement.BedContainer, jwtService *jwt.JwtService) {
	version := getVersion(app)
	bedGroup := version.Group("bed")

	// Apply JWT authentication middleware to all bed routes
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
func RegisterAuthRoute(app *fiber.App, service *authentication.UserService) {
	version := getVersion(app)
	authGroup := version.Group("authentication")
	auth := authentication.NewAuthController(service)
	authGroup.Post("/login", auth.Login)
	authGroup.Post("/refresh", auth.Refresh)
}
func RegisterDepartmentRoutes(app *fiber.App, service *department.DepartmentService, jwtservice *jwt.JwtService) {
	version := getVersion(app)
	departmentGrp := version.Group("department")
	departmentGrp.Use(func(c *fiber.Ctx) error {
		return middleware.Authenticate(c, jwtservice)
	})
	departmentController := department.NewDepartmentControllerInterface(service)
	departmentGrp.Get("/getDepartments", departmentController.FindMany)
}
func RegisterPermissionRoutes(app *fiber.App, service *permissions.PermService) {
	version := getVersion(app)
	permissionGrp := version.Group("permission")
	permissionGrp.Get("/getAll", func(c *fiber.Ctx) error {
		return permissions.FindMany(c, service)
	})
}
func RegisterPatientRoutes(app *fiber.App, service *patient.PatientService, jwtservice *jwt.JwtService) {
	version := getVersion(app)
	patientGroup := version.Group("patients")
	patientGroup.Use(func(c *fiber.Ctx) error {
		return middleware.Authenticate(c, jwtservice)
	})
	patientManagement := patient.NewPatientControllerInterface(service)
	patientGroup.Post("/addGeneralInfo", patientManagement.AddGeneralInfoHandler)
	patientGroup.Get("/getPatients", patientManagement.Find)
	patientGroup.Get("/getpatientByID/:patientID", patientManagement.GetPatientByID)

	//patientGroup.Post("/getPatients", patientManagement.PatientHandler)
}
func RegisterEmployeeRoutes(app *fiber.App, service *employee.EmployeeService) {
	version := getVersion(app)
	employeeGroup := version.Group("employee")
	employeeController := employee.NewEmployeeControllerInterface(service)
	employeeGroup.Post("/addEmployee", employeeController.Add)
	employeeGroup.Post("/create", employeeController.CreateAdmin)
	employeeGroup.Patch("/update", employeeController.UpdateUser)
	employeeGroup.Delete("/delete", employeeController.Delete)
	employeeGroup.Get("/findbyID", employeeController.FindByID)
	employeeGroup.Get("/getEmployees", employeeController.FindMany)
	employeeGroup.Get("/getDoctors", employeeController.FindDoctors)
}
func RegisterOrganisationRoutes(app *fiber.App, service *organisation.OrganisationService) {
	version := getVersion(app)
	organisationGrp := version.Group("organisation")
	organisationController := organisation.NewIOrganisationController(service)
	organisationGrp.Post("/signupOrg", organisationController.CreateOrganisation)
	organisationGrp.Patch("/updateLocation", organisationController.UpdateOrganisationLoc)
	organisationGrp.Get("/getbyid/:organisation_id", organisationController.GetByID)
	organisationGrp.Patch("/update", organisationController.Update)
}
func RegisterLicenseRoutes(app *fiber.App, service *license.LicenseService) {
	version := getVersion(app)
	licenseGrp := version.Group("license")
	licenseGrp.Patch("/verifylicense/:organisationID", func(c *fiber.Ctx) error {
		return license.VerifyLicense(c, service)
	})
}
func RegisterMedicineRoutes(app *fiber.App, service *medicine.MedicineService) {
	version := getVersion(app)
	medicineGrp := version.Group("medicine")
	medicineController := medicine.NewMedicineController(service)
	medicineGrp.Post("/addMedicine", medicineController.AddMedicine)
	medicineGrp.Get("/getMedicineByID", medicineController.GetByIDHandler)
	medicineGrp.Get("/GetMedicines", medicineController.GetAllHandler)
	medicineGrp.Get("/searchMedicine", medicineController.SearchMedicine)
}
func RegisterPrescriptionRoutes(app *fiber.App, service *prescription.PrescriptionService) {
	version := getVersion(app)
	prescriptionGrp := version.Group("prescription")
	prescriptionController := prescription.NewPrescriptionController(service)
	prescriptionGrp.Post("/create", prescriptionController.CreatePrescription)
	prescriptionGrp.Get("/get", prescriptionController.FindMany)
	prescriptionGrp.Patch("/update", prescriptionController.UpdatePrescription)
	prescriptionGrp.Get("/getprescriptionbyid/:prescription_id", prescriptionController.FindPrescriptionByID)
	prescriptionGrp.Get("/getPrescriptionByPatientID", prescriptionController.GetPrescriptionByPatientID)
	prescriptionGrp.Patch("/updateStatus", prescriptionController.UpdateStatus)
}
func RegisterSupplierRoutes(app *fiber.App, service *medicine.SupplierService) {
	version := getVersion(app)
	supplierGrp := version.Group("supplier")
	supplierController := medicine.NewSupplierController(service)
	supplierGrp.Get("/getSupplierByID", supplierController.GetSupplierByID)
	supplierGrp.Post("/createSupplier", supplierController.CreateSupplier)
}
func RegisterAppointments(app *fiber.App, service *appointments.AppointmentService) {
	version := getVersion(app)
	appointmentGrp := version.Group("appointment")
	appointmentController := appointments.NewAppointmentController(service)
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
	orgSchedController := admins.NewOrgSchedController(service)
	orgSched.Post("/create", orgSchedController.Create)

}
