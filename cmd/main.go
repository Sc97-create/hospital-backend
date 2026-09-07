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
	rbac := routers.RBACDeps{
		JWT:        containers.JwtManagement,
		RolePerm:   containers.RolePermissionService,
		RoleLookup: containers.EmployeeService,
	}
	routers.RegisterPatientRoutes(app, containers.PatientService, rbac)
	routers.RegisterOrganisationRoutes(app, containers.OrganisationService)
	routers.RegisterEmployeeRoutes(app, containers.EmployeeService, rbac)
	routers.RegisterMedicineRoutes(app, containers.MedContainer.Medicineservices, rbac)
	routers.RegisterAuthRoute(app, containers.AuthService, rbac)
	routers.RegisterPermissionRoutes(app, containers.PermissionService, rbac)
	routers.RegisterDepartmentRoutes(app, containers.DepartmentService, rbac)
	routers.RegisterRoleRoutes(app, containers.RoleService, rbac)
	routers.RegisterBedRoute(app, containers.BedManagement, containers.JwtManagement)
	routers.RegisterPrescriptionRoutes(app, containers.PrescriptionManagement, containers.PrescriptionItems, rbac)
	routers.RegisterSupplierRoutes(app, containers.MedContainer.SupplierService, rbac)
	routers.RegisterDashboardRoutes(app, containers.DashboardContainer.Service, rbac)
	routers.RegisterAppointments(app, containers.AppointmentContainer.Appointmentservice, rbac)
	routers.RegisterOrgSchedule(app, containers.OrganisationSchedule)
	routers.RegisterBillingRoutes(app, containers.BillingService, rbac)
	routers.RegisterPaymentRoutes(app, containers.PaymentContainer.Mod.Paymentservice, containers.PaymentContainer.Mod.WebhookService, rbac)
	err = app.Listen(fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		logger.Log.Fatal("server failed to start", zap.Error(err))
	}
}
