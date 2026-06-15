package appinit

import (
	"hospital-backend/config"
	"hospital-backend/internal/admins"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/authentication"
	"hospital-backend/internal/bedmanagement"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	jwtAuth "hospital-backend/internal/jwt"
	"hospital-backend/internal/license"
	"hospital-backend/internal/medicine/medcontainer"
	"hospital-backend/internal/modules"
	notificationcontainer "hospital-backend/internal/notifications/notificationcontianer"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/patient"
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
	OrganisationService    *organisation.OrganisationService
	AuthService            *authentication.UserService
	LicenseService         *license.LicenseService
	PermissionService      *permissions.PermService
	RolePermissionService  *rolepermissions.RolePermissionService
	DepartmentService      *department.DepartmentService
	ModuleService          *modules.ModuleService
	RoleService            *roles.RoleServices
	BedManagement          *bedmanagement.BedContainer
	JwtManagement          *jwtAuth.JwtService
	PrescriptionManagement *prescription.PrescriptionService
	MedContainer           *medcontainer.MedContainer
	AppointmentContainer   *appointments.AppntmentContainer
	OrganisationSchedule   *admins.OrganisationScheduleService
	NotificationContainer  *notificationcontainer.NotificationContainer
}

func NewContainer(db *gorm.DB, cfg *config.Config) *Container {
	bedmanagement := bedmanagement.NewBedContainer(db)
	medicineContainer := medcontainer.MedicineContainer(db)
	patientRepo := patient.NewPatientRepo(db)
	employeeRepo := employee.NewEmployeeRepo(db)
	jwtRepo := jwtAuth.NewRefreshTokenModel(db)
	organisationRepo := organisation.NewOrganisationRepo(db)
	jwtService := jwtAuth.NewJwtService(jwtRepo, cfg)
	authenticationRepo := authentication.NewAuthRepo(db)
	roleRepo := roles.NewRoleRepo(db)
	roleService := roles.NewRoleServices(roleRepo)
	DeptRepo := department.NewDepartmentRepo(db)
	deptService := department.NewDepartmentService(DeptRepo)
	licenseRepo := license.NewLicenseRepo(db)
	PermissionRepo := permissions.NewPermDB(db)
	ModuleRepo := modules.NewModuleDb(db)
	moduleService := modules.NewModuleService(ModuleRepo)
	permService := permissions.NewService(PermissionRepo, ModuleRepo)
	patientService := patient.NewPatientService(patientRepo)
	rolePermissionRepo := rolepermissions.NewRolePermissionDb(db)
	rolePermService := rolepermissions.NewRolePermissionService(db, rolePermissionRepo)
	departmentService := department.NewDepartmentService(DeptRepo)
	employeeService := employee.NewEmpService(db, employeeRepo, organisationRepo, roleService, deptService)
	authService := authentication.NewService(*authenticationRepo, *jwtService)
	licenseService := license.NewLicenseService(*licenseRepo)
	prescriptionRepo := prescription.NewPrescriptionDB(db)
	organisationSchedule := admins.NewCommonDB(db)
	orgschedSrv := admins.NewOrganisationScheduleService(organisationSchedule)
	notificationContainer := notificationcontainer.NewNotificationContainer(db, *cfg)
	appointmentSrv := appointments.AppointmentContainers(db, *orgschedSrv, notificationContainer.Service)
	prescriptionService := prescription.NewPrescriptionService(db, prescriptionRepo, medicineContainer.Medicineservices, appointmentSrv.Appointmentservice)
	orgService := organisation.NewOrganisationService(db, organisationRepo, licenseService, roleService, deptService, permService, rolePermService)

	return &Container{
		PatientService:         patientService,
		EmployeeService:        employeeService,
		AuthService:            &authService,
		OrganisationService:    orgService,
		LicenseService:         licenseService,
		MedContainer:           medicineContainer,
		PermissionService:      permService,
		DepartmentService:      departmentService,
		ModuleService:          moduleService,
		RoleService:            roleService,
		BedManagement:          bedmanagement,
		JwtManagement:          jwtService,
		PrescriptionManagement: prescriptionService,
		AppointmentContainer:   appointmentSrv,
		OrganisationSchedule:   orgschedSrv,
		NotificationContainer:  notificationContainer,
	}
}
