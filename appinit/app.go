package appinit

import (
	centralcontainer "hospital-backend/central/container"
	"hospital-backend/config"
	"hospital-backend/internal/admins"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/authentication"
	"hospital-backend/internal/bedmanagement"
	"hospital-backend/internal/billing"
	"hospital-backend/internal/dashboard"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	jwtAuth "hospital-backend/internal/jwt"
	"hospital-backend/internal/medicine/medcontainer"
	"hospital-backend/internal/modules"
	notificationcontainer "hospital-backend/internal/notifications/notificationcontianer"
	"hospital-backend/internal/orgbootstrap"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/payments/paymentcontainer"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/prescription"
	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/roles"

	"gorm.io/gorm"
)

type Container struct {
	PatientService  *patient.PatientService
	EmployeeService *employee.EmployeeService
	//EmailService        *email.EmailService
	Central                *centralcontainer.Container
	AuthService            *authentication.UserService
	PermissionService      *permissions.PermService
	RolePermissionService  *rolepermissions.RolePermissionService
	DepartmentService      *department.DepartmentService
	ModuleService          *modules.ModuleService
	RoleService            *roles.RoleServices
	BedManagement          *bedmanagement.BedContainer
	JwtManagement          *jwtAuth.JwtService
	PrescriptionItems      *prescription.PrescriptionItemServ
	PrescriptionManagement *prescription.PrescriptionService
	MedContainer           *medcontainer.MedContainer
	AppointmentContainer   *appointments.AppntmentContainer
	DashboardContainer     *dashboard.DashboardContainer
	OrganisationSchedule   *admins.OrganisationScheduleService
	NotificationContainer  *notificationcontainer.NotificationContainer
	PaymentContainer       *paymentcontainer.PaymentContainer
	BillingService         *billing.InvoiceServ
	OrgBootstrap           *orgbootstrap.Service
}

func NewContainer(db *gorm.DB, cfg *config.Config) *Container {
	bedmanagement := bedmanagement.NewBedContainer(db)
	medicineContainer := medcontainer.MedicineContainer(db)
	patientRepo := patient.NewPatientRepo(db)
	employeeRepo := employee.NewEmployeeRepo(db)
	jwtRepo := jwtAuth.NewRefreshTokenModel(db)
	jwtService := jwtAuth.NewJwtService(jwtRepo, cfg)
	authenticationRepo := authentication.NewAuthRepo(db)
	roleRepo := roles.NewRoleRepo(db)
	roleService := roles.NewRoleServices(roleRepo)
	DeptRepo := department.NewDepartmentRepo(db)
	deptService := department.NewDepartmentService(DeptRepo)
	PermissionRepo := permissions.NewPermDB(db)
	ModuleRepo := modules.NewModuleDb(db)
	moduleService := modules.NewModuleService(ModuleRepo)
	permService := permissions.NewService(PermissionRepo, ModuleRepo)

	rolePermissionRepo := rolepermissions.NewRolePermissionDb(db)
	rolePermService := rolepermissions.NewRolePermissionService(db, rolePermissionRepo)
	departmentService := department.NewDepartmentService(DeptRepo)
	notificationContainer := notificationcontainer.NewNotificationContainer(db, *cfg)
	central := centralcontainer.New(db, cfg, centralcontainer.Deps{
		Verifier:      jwtService,
		Notifications: notificationContainer.Service,
	})
	employeeService := employee.NewEmpService(db, employeeRepo, central.Organisations, roleService, deptService, notificationContainer.Service, cfg)
	authService := authentication.NewService(authenticationRepo, jwtService, rolePermService, notificationContainer.Service, cfg)
	prescriptionRepo := prescription.NewPrescriptionDB(db)
	organisationSchedule := admins.NewCommonDB(db)
	orgschedSrv := admins.NewOrganisationScheduleService(organisationSchedule)
	appointmentSrv := appointments.AppointmentContainers(db, orgschedSrv, notificationContainer.Service)
	prescriptionItemServ := prescription.NewPrescriptionItemService(prescriptionRepo)

	prescriptionService := prescription.NewPrescriptionService(db, prescriptionRepo, appointmentSrv.Appointmentservice, appointmentSrv.Appointmentservice, prescriptionItemServ, notificationContainer.Service)
	patientService := patient.NewPatientService(patientRepo, central.Organisations, notificationContainer.Service)
	billingRepo := billing.NewDB(db)
	billingItemServ := billing.NewInvoiceItemServ(billingRepo, prescriptionItemServ)
	fulfillment := newPaymentFulfillment(
		billingItemServ,
		billingRepo,
		medicineContainer.MedInventoryService,
		medicineContainer.MedMvmtService,
		prescriptionService,
		prescriptionItemServ,
		patientService,
		notificationContainer.Service,
	)
	paymentcontainer := paymentcontainer.NewContainer(db, *cfg, fulfillment, fulfillment, central.Subscriptions)
	billingService := billing.NewInvoiceServ(db, billingRepo, paymentcontainer.Mod.Paymentservice, billingItemServ, patientService, appointmentSrv.Appointmentservice)
	dashboardContainer := dashboard.NewDashboardContainer(appointmentSrv.Appointmentservice, billingService, employeeService, prescriptionService)
	orgSetup := orgbootstrap.New(db, roleService, rolePermService, permService, departmentService, employeeService)
	return &Container{
		PatientService:         patientService,
		EmployeeService:        employeeService,
		AuthService:            authService,
		Central:                central,
		MedContainer:           medicineContainer,
		PermissionService:      permService,
		DepartmentService:      departmentService,
		ModuleService:          moduleService,
		RoleService:            roleService,
		RolePermissionService:  rolePermService,
		BedManagement:          bedmanagement,
		JwtManagement:          jwtService,
		PrescriptionManagement: prescriptionService,
		AppointmentContainer:   appointmentSrv,
		DashboardContainer:     dashboardContainer,
		OrganisationSchedule:   orgschedSrv,
		NotificationContainer:  notificationContainer,
		PrescriptionItems:      prescriptionItemServ,
		PaymentContainer:       paymentcontainer,
		BillingService:         billingService,
		OrgBootstrap:           orgSetup,
	}
}
