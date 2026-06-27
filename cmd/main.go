package main

import (
	"fmt"
	"hospital-backend/appinit"
	"hospital-backend/config"
	"hospital-backend/database"
	"hospital-backend/pkg/middleware"
	"hospital-backend/pkg/middleware/routers"
	"hospital-backend/shared/migration"
	"log"
	"runtime"

	"github.com/gofiber/fiber/v2"
)

/*
error handling needs to handle everywhere
*/
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("%v", err)
	}
	migration.Migrate()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Println("HeapAlloc", m.HeapAlloc)
	fmt.Println("Heap Sys", m.HeapSys)
	fmt.Println("stack in use", m.StackInuse)
	clientConfig := fiber.Config{}
	clientConfig.AppName = "Hospital Management"
	clientConfig.CaseSensitive = true
	clientConfig.Concurrency = 256 * 1024
	clientConfig.DisableDefaultContentType = true
	clientConfig.EnableTrustedProxyCheck = true

	app := fiber.New(clientConfig)
	middleware.HandleMiddleware(app)
	containers := appinit.NewContainer(database.PostgreClient.GormDriver, cfg)
	err = containers.PermissionService.DefaultPerm()
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = containers.ModuleService.DefaultModule()
	if err != nil {
		log.Fatalf("%v", err)
	}
	routers.RegisterPatientRoutes(app, containers.PatientService, containers.JwtManagement)
	routers.RegisterOrganisationRoutes(app, containers.OrganisationService)
	routers.RegisterLicenseRoutes(app, containers.LicenseService)
	routers.RegisterEmployeeRoutes(app, containers.EmployeeService)
	routers.RegisterMedicineRoutes(app, containers.MedContainer.Medicineservices)
	routers.RegisterAuthRoute(app, containers.AuthService)
	routers.RegisterPermissionRoutes(app, containers.PermissionService)
	routers.RegisterDepartmentRoutes(app, containers.DepartmentService, containers.JwtManagement)
	routers.RegisterRoleRoutes(app, containers.RoleService)
	routers.RegisterBedRoute(app, containers.BedManagement, containers.JwtManagement)
	routers.RegisterPrescriptionRoutes(app, containers.PrescriptionManagement)
	routers.RegisterSupplierRoutes(app, containers.MedContainer.SupplierService)
	routers.RegisterAppointments(app, containers.AppointmentContainer.Appointmentservice)
	routers.RegisterOrgSchedule(app, containers.OrganisationSchedule)
	err = app.Listen(fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		log.Fatalf("%v", err)
	}
}
