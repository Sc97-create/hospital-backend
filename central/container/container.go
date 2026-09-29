package container

import (
	"hospital-backend/central/customers"
	"hospital-backend/central/internalapi"
	"hospital-backend/central/middleware"
	"hospital-backend/central/organisations"
	"hospital-backend/central/payments"
	"hospital-backend/central/plans"
	"hospital-backend/central/subscriptions"
	"hospital-backend/central/tenants"
	"hospital-backend/config"

	"gorm.io/gorm"
)

// Deps are hospital-side collaborators central does not own.
// Access tokens are minted by calling the internal JWT API.
type Deps struct {
	Verifier      middleware.TokenVerifier
	Notifications customers.NotificationEnqueuer
}

// Container wires central-db services. Hospital modules receive this container
// when they need an organisation or subscription collaborator.
type Container struct {
	Organisations *organisations.OrganisationService
	Tenants       *tenants.TenantService
	Plans         *plans.PlanService
	Internal      *internalapi.Client
	Payments      *payments.PaymentService
	Subscriptions *subscriptions.SubscriptionService
	Customers     *customers.CustomerService
	Verifier      middleware.TokenVerifier
}

func New(db *gorm.DB, cfg *config.Config, deps Deps) *Container {
	organisationRepo := organisations.NewOrganisationRepo(db)
	organisationService := organisations.NewOrganisationService(db, organisationRepo)
	planRepo := plans.NewPlanRepo(db)
	planService := plans.NewPlanService(planRepo)
	api := internalapi.New(cfg.InternalAPIBaseURL, cfg.InternalBasicAuth.ID, cfg.InternalBasicAuth.Secret)
	paymentService := payments.NewPaymentService(api)
	customerRepo := customers.NewCustomerRepo(db)
	tenantRepo := tenants.NewTenantRepo(db)
	tenantService := tenants.NewTenantService(db, tenantRepo, organisationService, customerRepo)
	subscriptionRepo := subscriptions.NewSubscriptionRepo(db)
	subscriptionService := subscriptions.NewSubscriptionService(
		db,
		subscriptionRepo,
		planRepo,
		paymentService,
		tenantService,
		cfg.RazorPayClient.RPayConfig.ApiKey,
	)
	customerService := customers.NewCustomerService(customerRepo, api, deps.Notifications)
	api.Customers = customerService
	organisationService.Admin = api
	tenantService.Admin = api
	return &Container{
		Organisations: organisationService,
		Tenants:       tenantService,
		Plans:         planService,
		Internal:      api,
		Payments:      paymentService,
		Subscriptions: subscriptionService,
		Customers:     customerService,
		Verifier:      deps.Verifier,
	}
}
