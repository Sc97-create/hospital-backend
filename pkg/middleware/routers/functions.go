package routers

import (
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
func RegisterDepartmentRoutes(app *fiber.App, service *department.DepartmentService) {
	version := getVersion(app)
	departmentGrp := version.Group("department")
	departmentController := department.NewDepartmentControllerInterface(service)
	departmentGrp.Get("/getDepartments/:organisation_id", departmentController.FindMany)
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
	//patientGroup.Post("/getPatients", patientManagement.PatientHandler)
}
func RegisterEmployeeRoutes(app *fiber.App, service *employee.EmployeeService) {
	version := getVersion(app)
	employeeGroup := version.Group("employee")
	employeeController := employee.NewEmployeeControllerInterface(service)
	employeeGroup.Post("/addEmployee", employeeController.AddHandler)
	employeeGroup.Post("/create", employeeController.CreateAdmin)
	employeeGroup.Patch("/update", employeeController.UpdateUser)
	employeeGroup.Delete("/delete", employeeController.DeleteHandler)
	employeeGroup.Get("/findbyID", employeeController.FindByIDHandler)
	employeeGroup.Get("/getEmployees", employeeController.FindManyHandler)
	employeeGroup.Get("/getDoctors", employeeController.FindDoctorsHandler)
}
func RegisterOrganisationRoutes(app *fiber.App, service *organisation.OrganisationService) {
	version := getVersion(app)
	organisationGrp := version.Group("organisation")
	organisationGrp.Post("/signupOrg", func(c *fiber.Ctx) error {
		return organisation.OrganisationSignup(c, service)
	})
	organisationGrp.Patch("/updateLocation", func(c *fiber.Ctx) error {
		return organisation.UpdateOrgLocation(c, service)
	})
	organisationGrp.Get("/getbyid/:organisation_id", func(c *fiber.Ctx) error {
		return organisation.GetByID(c, service)
	})
	organisationGrp.Patch("/update", func(c *fiber.Ctx) error {
		return organisation.Update(c, service)
	})
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
	medicineGrp.Post("/addMedicine", func(c *fiber.Ctx) error {
		return medicine.CreateHandler(c, service)
	})
	medicineGrp.Get("/getMedicineByID", func(c *fiber.Ctx) error {
		return medicine.GetByIDHandler(c, service)
	})
	medicineGrp.Get("/GetMedicines", func(c *fiber.Ctx) error {
		return medicine.GetAllHandler(c, service)
	})
}
