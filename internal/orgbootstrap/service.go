package orgbootstrap

import (
	"strings"

	"hospital-backend/central/customers"
	"hospital-backend/internal/department"
	empdto "hospital-backend/internal/employee/dto"
	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/roles"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type roleDirectory interface {
	FindRoleByOrgID(log *zap.Logger, organisationID string) ([]roles.Role, error)
	InsertMany(log *zap.Logger, tx *gorm.DB, organisationID string) ([]roles.Role, error)
	FindRoleByNames(log *zap.Logger, organisationID string, name string) (roles.Role, error)
}

type rolePermissionWriter interface {
	InsertMany(log *zap.Logger, tx *gorm.DB, roleArr []roles.Role, permissionsArr []permissions.Permission, modulesArr []modules.Modules, organisationID string) error
}

type permissionCatalog interface {
	FindMany(log *zap.Logger) ([]modules.Modules, []permissions.Permission, error)
}

type departmentDirectory interface {
	InsertMany(log *zap.Logger, tx *gorm.DB, organisationID string) error
	FindDeptByName(log *zap.Logger, organisationID string, name string) (department.Department, error)
}

type employeeInviter interface {
	CreateEmployeeInvite(log *zap.Logger, payload empdto.EmpRequest) (string, string, error)
}

type customerLookup interface {
	GetByTenantID(log *zap.Logger, tenantID string) (customers.Customer, error)
}

type Service struct {
	DB        *gorm.DB
	Roles     roleDirectory
	RolePerms rolePermissionWriter
	Catalog   permissionCatalog
	Depts     departmentDirectory
	Employees employeeInviter
	Customers customerLookup
}

func New(
	db *gorm.DB,
	roleSvc *roles.RoleServices,
	rolePerms *rolepermissions.RolePermissionService,
	catalog *permissions.PermService,
	depts *department.DepartmentService,
	employees employeeInviter,
) *Service {
	return &Service{
		DB:        db,
		Roles:     roleSvc,
		RolePerms: rolePerms,
		Catalog:   catalog,
		Depts:     depts,
		Employees: employees,
	}
}

type RoleResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FirstUserRequest struct {
	OrganisationID string
	FirstName      string
	LastName       string
	EmailID        string
	Username       string
	MobileNumber   string
	DateOfBirth    string
	DateOfJoining  string
}

type FirstUserResult struct {
	UserID           string `json:"user_id"`
	OrganisationID   string `json:"organisation_id"`
	RoleID           string `json:"role_id"`
	DepartmentID     string `json:"department_id"`
	TempPassword     string `json:"temp_password"`
	PasswordRequired bool   `json:"password_required"`
}

func (s *Service) AddRoles(log *zap.Logger, organisationID string) ([]RoleResult, error) {
	log = ensureLog(log)
	organisationID = strings.TrimSpace(organisationID)
	if organisationID == "" {
		return nil, wrapError.ErrInvalidRequest
	}
	existing, err := s.Roles.FindRoleByOrgID(log, organisationID)
	if err != nil {
		log.Error("organisation roles failed", zap.String("organisation_id", organisationID), zap.String("reason", "lookup"), zap.Error(err))
		return nil, wrapError.ErrOrganisationSetupFailed
	}
	if len(existing) > 0 {
		return mapRoles(existing), nil
	}
	created, err := s.Roles.InsertMany(log, s.DB, organisationID)
	if err != nil {
		log.Error("organisation roles failed", zap.String("organisation_id", organisationID), zap.String("reason", "insert"), zap.Error(err))
		return nil, wrapError.ErrOrganisationSetupFailed
	}
	log.Info("organisation roles success", zap.String("organisation_id", organisationID), zap.Int("count", len(created)))
	return mapRoles(created), nil
}

func (s *Service) AddRolePermissions(log *zap.Logger, organisationID string) error {
	log = ensureLog(log)
	organisationID = strings.TrimSpace(organisationID)
	if organisationID == "" {
		return wrapError.ErrInvalidRequest
	}
	rolesForOrg, err := s.Roles.FindRoleByOrgID(log, organisationID)
	if err != nil {
		log.Error("organisation role permissions failed", zap.String("organisation_id", organisationID), zap.String("reason", "roles"), zap.Error(err))
		return wrapError.ErrOrganisationSetupFailed
	}
	if len(rolesForOrg) == 0 {
		log.Warn("organisation role permissions failed", zap.String("organisation_id", organisationID), zap.String("reason", "roles_missing"))
		return wrapError.ErrInvalidRequest
	}
	count, err := s.countRolePermissions(organisationID)
	if err != nil {
		return err
	}
	if count > 0 {
		log.Info("organisation role permissions skipped", zap.String("organisation_id", organisationID), zap.String("reason", "already_present"))
		return nil
	}
	moduleRows, permissionRows, err := s.Catalog.FindMany(log)
	if err != nil {
		log.Error("organisation role permissions failed", zap.String("organisation_id", organisationID), zap.String("reason", "catalog"), zap.Error(err))
		return wrapError.ErrOrganisationSetupFailed
	}
	if len(moduleRows) == 0 || len(permissionRows) == 0 {
		log.Warn("organisation role permissions failed", zap.String("organisation_id", organisationID), zap.String("reason", "catalog_empty"))
		return wrapError.ErrOrganisationSetupFailed
	}
	if err := s.RolePerms.InsertMany(log, s.DB, rolesForOrg, permissionRows, moduleRows, organisationID); err != nil {
		log.Error("organisation role permissions failed", zap.String("organisation_id", organisationID), zap.String("reason", "insert"), zap.Error(err))
		return wrapError.ErrOrganisationSetupFailed
	}
	log.Info("organisation role permissions success", zap.String("organisation_id", organisationID))
	return nil
}

func (s *Service) AddFirstUser(log *zap.Logger, req FirstUserRequest) (FirstUserResult, error) {
	log = ensureLog(log)
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.EmailID = strings.TrimSpace(req.EmailID)
	req.Username = strings.TrimSpace(req.Username)
	req.MobileNumber = strings.TrimSpace(req.MobileNumber)
	req.DateOfBirth = strings.TrimSpace(req.DateOfBirth)
	req.DateOfJoining = strings.TrimSpace(req.DateOfJoining)
	if err := validateFirstUser(req); err != nil {
		log.Warn("organisation first user failed", zap.String("reason", "validation"))
		return FirstUserResult{}, wrapError.ErrInvalidRequest
	}
	if _, err := s.AddRoles(log, req.OrganisationID); err != nil {
		return FirstUserResult{}, err
	}
	if err := s.AddRolePermissions(log, req.OrganisationID); err != nil {
		return FirstUserResult{}, err
	}
	role, err := s.Roles.FindRoleByNames(log, req.OrganisationID, roles.DefaultRoleHospitalAdmin)
	if err != nil || strings.TrimSpace(role.ID) == "" {
		log.Error("organisation first user failed", zap.String("organisation_id", req.OrganisationID), zap.String("reason", "hospital_admin_role"), zap.Error(err))
		return FirstUserResult{}, wrapError.ErrOrganisationSetupFailed
	}
	deptID, err := s.ensureAdminDepartment(log, req.OrganisationID)
	if err != nil {
		return FirstUserResult{}, err
	}
	userID, tempPassword, err := s.Employees.CreateEmployeeInvite(log, empdto.EmpRequest{
		OrganisationID: req.OrganisationID,
		UserName:       req.Username,
		EmailID:        req.EmailID,
		RoleID:         role.ID,
		DepartmentID:   deptID,
		MobileNumber:   req.MobileNumber,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		DateOfBirth:    req.DateOfBirth,
		DateOfJoining:  req.DateOfJoining,
	})
	if err != nil {
		log.Error("organisation first user failed", zap.String("organisation_id", req.OrganisationID), zap.String("reason", "create_user"), zap.Error(err))
		return FirstUserResult{}, wrapError.ErrOrganisationSetupFailed
	}
	log.Info("organisation first user success",
		zap.String("organisation_id", req.OrganisationID),
		zap.String("user_id", userID),
		zap.String("role_id", role.ID),
	)
	return FirstUserResult{
		UserID:           userID,
		OrganisationID:   req.OrganisationID,
		RoleID:           role.ID,
		DepartmentID:     deptID,
		TempPassword:     tempPassword,
		PasswordRequired: true,
	}, nil
}

func (s *Service) ensureAdminDepartment(log *zap.Logger, organisationID string) (string, error) {
	dept, err := s.Depts.FindDeptByName(log, organisationID, department.DefaultDeptAdmin)
	if err != nil {
		log.Error("organisation first user failed", zap.String("organisation_id", organisationID), zap.String("reason", "department_lookup"), zap.Error(err))
		return "", wrapError.ErrOrganisationSetupFailed
	}
	if strings.TrimSpace(dept.ID) != "" {
		return dept.ID, nil
	}
	if err := s.Depts.InsertMany(log, s.DB, organisationID); err != nil {
		log.Error("organisation first user failed", zap.String("organisation_id", organisationID), zap.String("reason", "department_seed"), zap.Error(err))
		return "", wrapError.ErrOrganisationSetupFailed
	}
	dept, err = s.Depts.FindDeptByName(log, organisationID, department.DefaultDeptAdmin)
	if err != nil || strings.TrimSpace(dept.ID) == "" {
		log.Error("organisation first user failed", zap.String("organisation_id", organisationID), zap.String("reason", "department_missing"), zap.Error(err))
		return "", wrapError.ErrOrganisationSetupFailed
	}
	return dept.ID, nil
}

// ProvisionHospitalAdmin loads the customer for tenantID and creates them as Hospital Admin.
func (s *Service) ProvisionHospitalAdmin(log *zap.Logger, tenantID, organisationID string) error {
	log = ensureLog(log)
	if s.Customers == nil {
		log.Error("hospital admin provision failed", zap.String("reason", "customer_lookup_missing"))
		return wrapError.ErrOrganisationSetupFailed
	}
	customer, err := s.Customers.GetByTenantID(log, tenantID)
	if err != nil {
		log.Error("hospital admin provision failed",
			zap.String("tenant_id", tenantID),
			zap.String("organisation_id", organisationID),
			zap.String("reason", "customer_lookup"),
			zap.Error(err),
		)
		return err
	}
	req := firstUserFromCustomer(organisationID, customer)
	if _, err = s.AddFirstUser(log, req); err != nil {
		log.Error("hospital admin provision failed",
			zap.String("tenant_id", tenantID),
			zap.String("organisation_id", organisationID),
			zap.String("customer_id", customer.ID),
			zap.String("reason", "first_user"),
			zap.Error(err),
		)
		return err
	}
	log.Info("hospital admin provision success",
		zap.String("tenant_id", tenantID),
		zap.String("organisation_id", organisationID),
		zap.String("customer_id", customer.ID),
	)
	return nil
}

func firstUserFromCustomer(organisationID string, customer customers.Customer) FirstUserRequest {
	firstName, lastName := splitFullName(customer.FullName)
	return FirstUserRequest{
		OrganisationID: organisationID,
		FirstName:      firstName,
		LastName:       lastName,
		EmailID:        customer.WorkEmail,
		Username:       usernameFromEmail(customer.WorkEmail),
	}
}

func splitFullName(fullName string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(fullName))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func usernameFromEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	local, _, found := strings.Cut(email, "@")
	if !found || local == "" {
		return email
	}
	return local
}

func (s *Service) countRolePermissions(organisationID string) (int64, error) {
	if s.DB == nil {
		return 0, wrapError.ErrOrganisationSetupFailed
	}
	var count int64
	err := s.DB.Model(&rolepermissions.RolePermission{}).Where("organisation_id = ?", organisationID).Count(&count).Error
	if err != nil {
		return 0, wrapError.ErrOrganisationSetupFailed
	}
	return count, nil
}

func validateFirstUser(req FirstUserRequest) error {
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.EmailID = strings.TrimSpace(req.EmailID)
	req.Username = strings.TrimSpace(req.Username)
	req.DateOfBirth = strings.TrimSpace(req.DateOfBirth)
	if req.OrganisationID == "" || req.FirstName == "" || req.EmailID == "" || req.Username == "" {
		return wrapError.ErrInvalidRequest
	}
	return nil
}

func mapRoles(rows []roles.Role) []RoleResult {
	out := make([]RoleResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, RoleResult{ID: row.ID, Name: row.Name})
	}
	return out
}

func ensureLog(log *zap.Logger) *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}
	return log
}
