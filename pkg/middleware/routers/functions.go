package routers

import (
	"hospital-backend/internal/authentication"
	"hospital-backend/internal/bedmanagement"
	bedcontroller "hospital-backend/internal/bedmanagement/controllers"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/license"
	"hospital-backend/internal/medicine"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/roles"
	"log"

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
func RegisterBedRoute(app *fiber.App, services *bedmanagement.BedContainer) {
	version := getVersion(app)
	bedGroup := version.Group("bed")
	RoomTypeManagement := bedcontroller.NewRoomtypeControllerInterface(services.RoomTypeService)
	RoomManagement := bedcontroller.NewRoomControllerInterface(services.RoomServices)
	BedManagement := bedcontroller.NewBedControllerInterface(services.BedServices)

	bedGroup.Post("/createRoomType", RoomTypeManagement.CreateRoomTypeController)
	bedGroup.Get("/getRoomTypeData", RoomTypeManagement.GetRoomTypeData)
	bedGroup.Post("/createRoom", RoomManagement.CreateRoomController)
	bedGroup.Post("/createBed", BedManagement.CreateBedController)
	bedGroup.Post("/generateBeds", BedManagement.GenerateBeds)

}
func RegisterAuthRoute(app *fiber.App, service *authentication.UserService) {
	version := getVersion(app)
	authGroup := version.Group("authentication")
	authGroup.Post("/login", func(c *fiber.Ctx) error {
		return authentication.Login(c, service)
	})
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
func RegisterPatientRoutes(app *fiber.App, service *patient.PatientService) {
	version := getVersion(app)
	patientGroup := version.Group("patient")
	patientGroup.Post("/addGeneralInfo", func(c *fiber.Ctx) error {
		err := patient.AddGeneralInfoHandler(c, service)
		if err != nil {
			log.Println("err", err)
			return err
		}
		return nil
	})
	patientGroup.Post("/getPatients", func(c *fiber.Ctx) error {
		return patient.PatientHandler(c, service)
	})
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
