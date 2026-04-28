package main

import (
	"fmt"
	"hospital-backend/appinit"
	"hospital-backend/config"
	lconfig "hospital-backend/config"
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

func init() {
	lconfig.Init()
	migration.Migrate()
}
func main() {
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
	containers := appinit.NewContainer(config.PostgreClient.GormDriver)
	err := containers.PermissionService.DefaultPerm()
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
	routers.RegisterMedicineRoutes(app, containers.MedicineService)
	routers.RegisterAuthRoute(app, containers.AuthService)
	routers.RegisterPermissionRoutes(app, containers.PermissionService)
	routers.RegisterDepartmentRoutes(app, containers.DepartmentService)
	routers.RegisterRoleRoutes(app, containers.RoleService)
	routers.RegisterBedRoute(app, containers.BedManagement, containers.JwtManagement)
	err = app.Listen(":9069")
	if err != nil {
		log.Fatalf("%v", err)
	}
}
