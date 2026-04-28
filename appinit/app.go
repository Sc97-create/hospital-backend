package appinit

import (
	"hospital-backend/internal/authentication"
	"hospital-backend/internal/bedmanagement"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	jwtAuth "hospital-backend/internal/jwt"
	"hospital-backend/internal/license"
	"hospital-backend/internal/medicine"
	"hospital-backend/internal/modules"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/roles"

	"gorm.io/gorm"
)

type Container struct {
	PatientService  *patient.PatientService
	EmployeeService *employee.EmployeeService
	//EmailService        *email.EmailService
	OrganisationService   *organisation.OrganisationService
	AuthService           *authentication.UserService
	LicenseService        *license.LicenseService
	MedicineService       *medicine.MedicineService
	PermissionService     *permissions.PermService
	RolePermissionService *rolepermissions.RolePermissionService
	DepartmentService     *department.DepartmentService
	ModuleService         *modules.ModuleService
	RoleService           *roles.RoleServices
	BedManagement         *bedmanagement.BedContainer
	JwtManagement         *jwtAuth.JwtService
}

func NewContainer(db *gorm.DB) *Container {
	bedmanagement := bedmanagement.NewBedContainer(db)
	patientRepo := patient.NewPatientRepo(db)
	employeeRepo := employee.NewEmployeeRepo(db)
	jwtRepo := jwtAuth.NewRefreshTokenModel(db)
	organisationRepo := organisation.NewOrganisationRepo(db)
	jwtService := jwtAuth.NewJwtService(jwtRepo)
	authenticationRepo := authentication.NewAuthRepo(db)
	roleRepo := roles.NewRoleRepo(db)
	DeptRepo := department.NewDepartmentRepo(db)
	licenseRepo := license.NewLicenseRepo(db)
	medicineRepo := medicine.NewMedicineRepo(db)
	PermissionRepo := permissions.NewPermDB(db)
	ModuleRepo := modules.NewModuleDb(db)
	moduleService := modules.NewModuleService(ModuleRepo)
	permService := permissions.NewService(PermissionRepo, ModuleRepo)
	patientService := patient.NewPatientService(patientRepo)
	rolePermissionRepo := rolepermissions.NewRolePermissionDb(db)
	departmentService := department.NewDepartmentService(DeptRepo)
	employeeService := employee.NewEmpService(db, employeeRepo, organisationRepo, roleRepo, DeptRepo, rolePermissionRepo, PermissionRepo)
	authService := authentication.NewService(*authenticationRepo, *jwtService)
	licenseService := license.NewLicenseService(*licenseRepo)
	medicineService := medicine.NewMedicineService(*medicineRepo)
	roleService := roles.NewRoleServices(roleRepo)
	orgService := organisation.NewOrganisationService(db, organisationRepo, licenseService)
	return &Container{
		PatientService:      patientService,
		EmployeeService:     employeeService,
		AuthService:         &authService,
		OrganisationService: orgService,
		LicenseService:      licenseService,
		MedicineService:     medicineService,
		PermissionService:   permService,
		DepartmentService:   departmentService,
		ModuleService:       moduleService,
		RoleService:         roleService,
		BedManagement:       bedmanagement,
		JwtManagement:       jwtService,
	}
}
