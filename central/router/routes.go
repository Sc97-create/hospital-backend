package router

import (
	"hospital-backend/central/container"
	"hospital-backend/central/customers"
	"hospital-backend/central/middleware"
	"hospital-backend/central/organisations"
	"hospital-backend/central/plans"
	"hospital-backend/central/subscriptions"
	"hospital-backend/central/tenants"
	"hospital-backend/config"
	pkgmiddleware "hospital-backend/pkg/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register mounts central APIs under /api/v1/central.
// Auth is central JWT only; hospital RBAC does not apply.
func Register(app *fiber.App, c *container.Container) {
	version := app.Group("api").Group("v1")
	auth := middleware.Authenticate(c.Verifier)

	registerOrganisationRoutes(version, c.Organisations)
	registerTenantRoutes(version, c.Tenants, auth)
	registerPlanRoutes(version, c.Plans, auth)
	registerSubscriptionRoutes(version, c.Subscriptions, auth)
	registerCustomerRoutes(version, c.Customers, auth)
}

// RegisterCheckEnd mounts the subscription end check for the hospital service to call.
// It is separate from the JWT subscription routes and uses Basic Auth.
func RegisterCheckEnd(app *fiber.App, service *subscriptions.SubscriptionService, basicAuth config.InternalBasicAuth) {
	ctrl := subscriptions.NewISubscriptionController(service)
	grp := app.Group("api").Group("v1").Group("central/internal/subscription")
	grp.Use(func(c *fiber.Ctx) error {
		return pkgmiddleware.AuthenticateBasic(c, basicAuth.ID, basicAuth.Secret)
	})
	grp.Get("/checkEnd/:organisation_id", ctrl.CheckEnd)
}

func registerOrganisationRoutes(version fiber.Router, service *organisations.OrganisationService) {
	grp := version.Group("central/organisation")
	ctrl := organisations.NewIOrganisationController(service)
	grp.Post("/add", ctrl.AddOrganisation)
	grp.Patch("/updateAddress", ctrl.UpdateAddress)
	grp.Get("/getbyid/:organisation_id", ctrl.GetByID)
	grp.Get("/listByTenant/:tenant_id", ctrl.ListByTenant)
	grp.Patch("/update", ctrl.Update)
}

func registerTenantRoutes(version fiber.Router, service *tenants.TenantService, auth fiber.Handler) {
	grp := version.Group("central/tenant")
	grp.Use(auth)
	ctrl := tenants.NewITenantController(service)
	grp.Post("/create", ctrl.Create)
	grp.Get("/getById/:tenant_id", ctrl.GetByID)
	grp.Patch("/update", ctrl.Update)
}

func registerPlanRoutes(version fiber.Router, service *plans.PlanService, auth fiber.Handler) {
	grp := version.Group("central/plan")
	grp.Use(auth)
	ctrl := plans.NewIPlanController(service)
	grp.Get("/list", ctrl.List)
}

func registerSubscriptionRoutes(version fiber.Router, service *subscriptions.SubscriptionService, auth fiber.Handler) {
	grp := version.Group("central/subscription")
	grp.Use(auth)
	ctrl := subscriptions.NewISubscriptionController(service)
	grp.Post("/create", ctrl.Create)
	grp.Post("/confirmCheckout", ctrl.ConfirmCheckout)
}

func registerCustomerRoutes(version fiber.Router, service *customers.CustomerService, auth fiber.Handler) {
	ctrl := customers.NewICustomerController(service)
	version.Group("central/customer").Post("/signup", ctrl.Signup)
	verify := version.Group("central/customer")
	verify.Use(auth)
	verify.Post("/verifyEmail", ctrl.VerifyEmail)
}
