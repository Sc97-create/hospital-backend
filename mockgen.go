package hospitalbackend

// Service interface mocks for controller unit tests.
// Repository interface mocks for service-layer unit tests.
// Regenerate all: go generate ./mockgen.go

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/authentication/mocks/mock_auth_servicer.go -package=mocks hospital-backend/internal/authentication AuthServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/billing/mocks/mock_invoice_servicer.go -package=mocks hospital-backend/internal/billing InvoiceServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/patient/mocks/mock_patient_servicer.go -package=mocks hospital-backend/internal/patient PatientServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/department/mocks/mock_department_servicer.go -package=mocks hospital-backend/internal/department DepartmentServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/roles/mocks/mock_role_servicer.go -package=mocks hospital-backend/internal/roles RoleServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/rolepermissions/mocks/mock_role_permission_servicer.go -package=mocks hospital-backend/internal/rolepermissions RolePermissionServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/permissions/mocks/mock_permission_servicer.go -package=mocks hospital-backend/internal/permissions PermissionServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/employee/mocks/mock_employee_servicer.go -package=mocks hospital-backend/internal/employee EmployeeServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/organisation/mocks/mock_organisation_servicer.go -package=mocks hospital-backend/internal/organisation OrganisationServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/payments/mocks/mock_payment_servicer.go -package=mocks hospital-backend/internal/payments PaymentServicer,WebhookServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/prescription/mocks/mock_prescription_servicer.go -package=mocks hospital-backend/internal/prescription PrescriptionServicer,PrescriptionItemServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/medicine/mocks/mock_medicine_servicer.go -package=mocks hospital-backend/internal/medicine MedicineServicer,SupplierServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/admins/mocks/mock_organisation_schedule_servicer.go -package=mocks hospital-backend/internal/admins OrganisationScheduleServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/appointments/mocks/mock_appointment_servicer.go -package=mocks hospital-backend/internal/appointments AppointmentServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/dashboard/mocks/mock_dashboard_servicer.go -package=mocks hospital-backend/internal/dashboard DashboardServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/dashboard/mocks/mock_appointment_dashboard_reader.go -package=mocks hospital-backend/internal/dashboard AppointmentDashboardReader
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/dashboard/mocks/mock_billing_dashboard_reader.go -package=mocks hospital-backend/internal/dashboard BillingDashboardReader
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/dashboard/mocks/mock_employee_dashboard_reader.go -package=mocks hospital-backend/internal/dashboard EmployeeDashboardReader
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/dashboard/mocks/mock_prescription_dashboard_reader.go -package=mocks hospital-backend/internal/dashboard PrescriptionDashboardReader

// Repository interface mocks for service-layer unit tests.

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/authentication/mocks/mock_user_repository.go -package=mocks hospital-backend/internal/authentication UserRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/authentication/mocks/mock_jwt_servicer.go -package=mocks hospital-backend/internal/authentication JwtServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/authentication/mocks/mock_role_permission_servicer.go -package=mocks hospital-backend/internal/authentication RolePermissionServicer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/authentication/mocks/mock_notification_enqueuer.go -package=mocks hospital-backend/internal/authentication NotificationEnqueuer
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/billing/mocks/mock_invoice_repository.go -package=mocks hospital-backend/internal/billing InvoiceRepo,InvoiceItemRepo
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/patient/mocks/mock_patient_repository.go -package=mocks hospital-backend/internal/patient PatientRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/department/mocks/mock_department_repository.go -package=mocks hospital-backend/internal/department DepartmentRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/roles/mocks/mock_role_repository.go -package=mocks hospital-backend/internal/roles RoleRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/permissions/mocks/mock_permission_repository.go -package=mocks hospital-backend/internal/permissions PermissionRepo
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/employee/mocks/mock_employee_repository.go -package=mocks hospital-backend/internal/employee EmployeeRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/organisation/mocks/mock_signup_deps.go -package=mocks hospital-backend/internal/organisation PermissionCatalogLookup,RoleSeeder,DepartmentSeeder,RolePermissionSeeder
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/organisation/mocks/mock_organisation_repository.go -package=mocks hospital-backend/internal/organisation OrganisationRepo
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/payments/mocks/mock_payments_repository.go -package=mocks hospital-backend/internal/payments IPaymentsRepository,IWebhookRepository,IPaymentAttempts
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/payments/mocks/mock_payment_fulfillment.go -package=mocks hospital-backend/internal/payments IPaymentFulfillment,PaymentAttemptServicer,FulfillmentServicer,PaymentInvoiceLookup,PrescriptionStatusUpdater
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/payments/providers/mocks/mock_provider.go -package=mocks hospital-backend/internal/payments/providers Provider,IPaymentFactory
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/prescription/mocks/mock_prescription_repository.go -package=mocks hospital-backend/internal/prescription PrescriptionRepositoryInterface,PrescItemsRepo
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/prescription/mocks/mock_prescription_deps.go -package=mocks hospital-backend/internal/prescription NotificationEnqueuer,AppointmentLookup,AppointmentStatusUpdater,PrescriptionItemAdder
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/medicine/mocks/mock_medicine_repository.go -package=mocks hospital-backend/internal/medicine MedicineRepository,ISupplier,RMedicineInventory,RPurchaseEntry,RMedicineMvmt
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/rolepermissions/mocks/mock_role_permission_repository.go -package=mocks hospital-backend/internal/rolepermissions RolePermissionRepo
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/appointments/mocks/mock_appointment_repository.go -package=mocks hospital-backend/internal/appointments AppointmentRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/admins/mocks/mock_organisation_schedule_repository.go -package=mocks hospital-backend/internal/admins OrganisationScheduleRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/modules/mocks/mock_module_repository.go -package=mocks hospital-backend/internal/modules ModuleRepo
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/jwt/mocks/mock_refresh_token_repository.go -package=mocks hospital-backend/internal/jwt RefreshtokenRepo
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/notifications/mocks/mock_notification_repository.go -package=mocks hospital-backend/internal/notifications/repository Repository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=internal/bedmanagement/mocks/mock_bedmanagement_repository.go -package=mocks hospital-backend/internal/bedmanagement/repository BedRepository,BedAllotmentRepository,RoomRepository,RoomTypeRepository,RoomSummaryRepository
