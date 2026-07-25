package main

import (
	"context"
	"fmt"
	"hospital-backend/appinit"
	"hospital-backend/config"
	"hospital-backend/database"
	"hospital-backend/pkg/logger"
	"hospital-backend/pkg/middleware"
	"hospital-backend/pkg/middleware/routers"
	"hospital-backend/shared/migration"
	"log"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

/*
error handling needs to handle everywhere
*/
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("%v", err)
	}

	logger.Init(cfg.Env, cfg.LogLevel)
	defer logger.Sync()

	logger.Log.Info("starting hospital backend",
		zap.String("env", cfg.Env),
		zap.String("log_level", cfg.LogLevel),
	)

	err = database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Log.Fatal("database connection failed", zap.Error(err))
	}

	migration.Migrate()
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
		logger.Log.Fatal("failed to seed default permissions", zap.Error(err))
	}
	err = containers.ModuleService.DefaultModule()
	if err != nil {
		logger.Log.Fatal("failed to seed default modules", zap.Error(err))
	}
	ctx := context.Background()
	containers.NotificationContainer.Start(ctx)
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
	routers.RegisterPrescriptionRoutes(app, containers.PrescriptionManagement, containers.PrescriptionItems)
	routers.RegisterSupplierRoutes(app, containers.MedContainer.SupplierService)
	routers.RegisterAppointments(app, containers.AppointmentContainer.Appointmentservice)
	routers.RegisterOrgSchedule(app, containers.OrganisationSchedule)
	routers.RegisterBillingRoutes(app, containers.BillingService)
	routers.RegisterPaymentRoutes(app, containers.PaymentContainer.Mod.Paymentservice, containers.PaymentContainer.Mod.WebhookService)
	err = app.Listen(fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		logger.Log.Fatal("server failed to start", zap.Error(err))
	}
}
