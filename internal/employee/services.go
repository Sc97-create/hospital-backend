package employee

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/central/organisations"
	"hospital-backend/config"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee/dto"
	"hospital-backend/internal/employee/utils"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/roles"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type NotificationEnqueuer interface {
	Create(ctx context.Context, data notificationdto.CreateRequest) error
}

type EmployeeService struct {
	DB                   *gorm.DB
	EmpRepo              EmployeeRepository
	OrganisationServicer organisations.OrganisationServicer
	RoleServices         *roles.RoleServices
	DeptServices         *department.DepartmentService
	Notifications        NotificationEnqueuer
	cfg                  *config.Config
}

func NewEmpService(db *gorm.DB, empRepo EmployeeRepository, orgService organisations.OrganisationServicer, roleServices *roles.RoleServices, deptServices *department.DepartmentService, notifications NotificationEnqueuer, cfg *config.Config) *EmployeeService {
	return &EmployeeService{DB: db, EmpRepo: empRepo, OrganisationServicer: orgService, RoleServices: roleServices, DeptServices: deptServices, Notifications: notifications, cfg: cfg}
}

func (EService *EmployeeService) CreateEmployee(log *zap.Logger, payload dto.EmpRequest) (id string, err error) {
	id, _, err = EService.CreateEmployeeInvite(log, payload)
	return id, err
}

// CreateEmployeeInvite creates a staff user with an empty password hash and a temp password.
// Login with the temp password sets password_cleared so the client can call updatePasswordFirstLogin.
func (EService *EmployeeService) CreateEmployeeInvite(log *zap.Logger, payload dto.EmpRequest) (id string, tempPassword string, err error) {
	log = ensureLog(log)
	tempPassword = utils.CreateTempPassword(payload.FirstName, payload.DateOfBirth)
	user := EService.toEmpModel([]byte{}, payload, payload.RoleID, payload.DepartmentID)
	user.TempPassword = tempPassword
	code, err := EService.createEmployeeCode(log, payload.OrganisationID, payload.DateOfJoining)
	if err != nil {
		if errors.Is(err, wrapError.ErrInvalidRequest) {
			return "", "", err
		}
		return "", "", wrapError.ErrEmployeeCreateFailed
	}
	user.EmployeeCode = code
	if err = EService.EmpRepo.Create(log, &user); err != nil {
		log.Error("employee create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", "", wrapError.ErrEmployeeCreateFailed
	}
	log.Info("employee create success",
		zap.String("user_id", user.ID),
		zap.String("organisation_id", payload.OrganisationID),
		zap.String("role_id", payload.RoleID),
	)
	EService.enqueueEmployeeCreated(log, user)
	return user.ID, tempPassword, nil
}

func (Eservice *EmployeeService) DeleteEmployee(log *zap.Logger, userID string) (err error) {
	log = ensureLog(log)
	if userID == "" {
		log.Warn("employee delete failed", zap.String("reason", "missing_user_id"))
		return wrapError.ErrInvalidRequest
	}
	err = Eservice.EmpRepo.DeleteOne(log, userID)
	if err != nil {
		log.Error("employee delete failed",
			zap.String("user_id", userID),
			zap.String("reason", "db_delete"),
			zap.Error(err),
		)
		return wrapError.ErrEmployeeDeleteFailed
	}
	log.Info("employee delete success", zap.String("user_id", userID))
	return
}
func (Eservice *EmployeeService) FindOne(log *zap.Logger, id string) (dto.EmployeeResponse, error) {
	log = ensureLog(log)
	row, err := Eservice.EmpRepo.ReadOne(log, id)
	if err != nil {
		if errors.Is(err, ErrEmployeeNotFound) {
			log.Warn("employee get failed", zap.String("user_id", id), zap.String("reason", "not_found"))
			return dto.EmployeeResponse{}, err
		}
		log.Error("employee get failed", zap.String("user_id", id), zap.String("reason", "db_read"), zap.Error(err))
		return dto.EmployeeResponse{}, wrapError.ErrEmployeeFetchFailed
	}
	return Eservice.mapToEmployeeResponse(*row), nil
}
func (Eservice *EmployeeService) FindMany(log *zap.Logger, req dto.FindManyRequest) (employeeResp []dto.EmployeeResponse, total int64, err error) {
	log = ensureLog(log)
	req.Search = strings.TrimSpace(req.Search)
	limitInt, skip := Eservice.getPageSkip(req.Limit, req.PageNo)
	users, err := Eservice.EmpRepo.ReadMany(log, limitInt, skip, req.OrganisationID, req.Search)
	if err != nil {
		log.Error("employee list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_list"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrEmployeesFetchFailed
	}
	total, err = Eservice.EmpRepo.Count(log, req.OrganisationID, req.Search)
	if err != nil {
		log.Error("employee list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrEmployeesFetchFailed
	}
	employeeResp = Eservice.arrayMapToEmployeeResponse(users)
	return
}

func (Eservice *EmployeeService) GetEmployeeStatusCounts(log *zap.Logger, organisationID string) (dto.EmployeeStatusCounts, error) {
	if strings.TrimSpace(organisationID) == "" {
		return dto.EmployeeStatusCounts{}, wrapError.ErrInvalidRequest
	}
	log = ensureLog(log)
	row, err := Eservice.EmpRepo.CountByActiveStatus(log, organisationID)
	if err != nil {
		log.Error("employee status counts failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.EmployeeStatusCounts{}, wrapError.ErrEmployeesFetchFailed
	}
	return dto.EmployeeStatusCounts{
		Active:   row.Active,
		Inactive: row.Inactive,
		Total:    row.Active + row.Inactive,
	}, nil
}

func (Eservice *EmployeeService) getPageSkip(limit int, pageNo int) (int, int) {
	skip := 0
	if pageNo != 0 {
		skip = (pageNo - 1) * limit
	}
	return limit, skip
}

func (Eservice *EmployeeService) arrayMapToEmployeeResponse(rows []EmployeeListRow) []dto.EmployeeResponse {
	employeeResponse := []dto.EmployeeResponse{}
	for _, each := range rows {
		employeeResponse = append(employeeResponse, Eservice.mapToEmployeeResponse(each))
	}
	return employeeResponse
}

func (Eservice *EmployeeService) mapToEmployeeResponse(row EmployeeListRow) dto.EmployeeResponse {
	name := row.Username
	if name == "" {
		name = strings.TrimSpace(row.FirstName + " " + row.LastName)
	}
	status := "inactive"
	if row.IsActive {
		status = "active"
	}
	return dto.EmployeeResponse{
		EmployeeID:             row.ID,
		EmployeeCode:           row.EmployeeCode,
		EmployeeName:           name,
		EmployeeFirstName:      row.FirstName,
		EmployeeLastName:       row.LastName,
		EmployeeEmail:          row.EmailID,
		EmployeePhone:          row.PhoneNumber,
		EmployeeRoleID:         row.RoleID,
		RoleName:               row.RoleName,
		EmployeeDepartmentID:   row.DepartmentID,
		DepartmentName:         row.DepartmentName,
		EmployeeStatus:         status,
		EmployeeOrganisationID: row.OrganisationID,
	}
}

func (Eservice *EmployeeService) FindRoleIDByUserID(log *zap.Logger, userID string) (string, error) {
	log = ensureLog(log)
	roleID, err := Eservice.EmpRepo.FindRoleIDByUserID(log, userID)
	if err != nil {
		if errors.Is(err, ErrEmployeeNotFound) {
			log.Warn("employee role lookup failed", zap.String("user_id", userID), zap.String("reason", "not_found"))
			return "", err
		}
		log.Error("employee role lookup failed", zap.String("user_id", userID), zap.String("reason", "db_read"), zap.Error(err))
		return "", wrapError.ErrEmployeeFetchFailed
	}
	return roleID, nil
}

func (Eservice *EmployeeService) FindOrganisationIDByUserID(log *zap.Logger, userID string) (string, error) {
	log = ensureLog(log)
	organisationID, err := Eservice.EmpRepo.FindOrganisationIDByUserID(log, userID)
	if err != nil {
		if errors.Is(err, ErrEmployeeNotFound) {
			log.Warn("employee organisation lookup failed", zap.String("user_id", userID), zap.String("reason", "not_found"))
			return "", err
		}
		log.Error("employee organisation lookup failed", zap.String("user_id", userID), zap.String("reason", "db_read"), zap.Error(err))
		return "", wrapError.ErrEmployeeFetchFailed
	}
	return organisationID, nil
}
func (Eservice *EmployeeService) CreateAdminProf(log *zap.Logger, payload dto.EmpRequest) (userID string, err error) {
	log = ensureLog(log)
	passwordHash, err := Eservice.hashPassword(payload.Password)
	if err != nil {
		log.Error("employee admin create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", "hash_password"),
		)
		return "", wrapError.ErrEmployeeCreateFailed
	}
	role, err := Eservice.RoleServices.FindRoleByNames(log, payload.OrganisationID, roles.DefaultRoleAdmin)
	if err != nil {
		log.Error("employee admin create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", "role_lookup"),
			zap.Error(err),
		)
		return "", wrapError.ErrEmployeeCreateFailed
	}
	department, err := Eservice.DeptServices.FindDeptByName(log, payload.OrganisationID, department.DefaultDeptAdmin)
	if err != nil {
		log.Error("employee admin create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", "department_lookup"),
			zap.Error(err),
		)
		return "", wrapError.ErrEmployeeCreateFailed
	}
	user := Eservice.toEmpModel(passwordHash, payload, role.ID, department.ID)
	code, err := Eservice.createEmployeeCode(log, payload.OrganisationID, payload.DateOfJoining)
	if err != nil {
		if errors.Is(err, wrapError.ErrInvalidRequest) {
			return "", err
		}
		return "", wrapError.ErrEmployeeCreateFailed
	}
	user.EmployeeCode = code
	err = Eservice.EmpRepo.Create(log, &user)
	if err != nil {
		log.Error("employee admin create failed",
			zap.String("organisation_id", payload.OrganisationID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", wrapError.ErrEmployeeCreateFailed
	}
	log.Info("employee admin create success",
		zap.String("user_id", user.ID),
		zap.String("organisation_id", payload.OrganisationID),
	)
	return user.ID, nil
}
func (Eservice *EmployeeService) UpdateAdminProf(log *zap.Logger, payload dto.UpdateRequest) (err error) {
	log = ensureLog(log)
	userData, err := Eservice.EmpRepo.ReadOne(log, payload.UserID)
	if err != nil {
		if errors.Is(err, ErrEmployeeNotFound) {
			log.Warn("employee update failed", zap.String("user_id", payload.UserID), zap.String("reason", "not_found"))
			return ErrEmployeeNotFound
		}
		log.Error("employee update failed",
			zap.String("user_id", payload.UserID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return wrapError.ErrEmployeeUpdateFailed
	}
	updateUser := make(map[string]interface{})
	if userData.FirstName != payload.FirstName {
		updateUser["first_name"] = payload.FirstName
	}
	if userData.LastName != payload.LastName {
		updateUser["last_name"] = payload.LastName
	}
	if payload.Password != "" {
		passwordHash, hashErr := Eservice.hashPassword(payload.Password)
		if hashErr != nil {
			return wrapError.ErrEmployeeUpdateFailed
		}
		updateUser["password_hash"] = string(passwordHash)
	}
	if payload.FirstName != "" && payload.LastName != "" {
		updateUser["username"] = payload.FirstName + " " + payload.LastName
	}
	if err = Eservice.EmpRepo.Update(log, payload.UserID, updateUser); err != nil {
		log.Error("employee update failed",
			zap.String("user_id", payload.UserID),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrEmployeeUpdateFailed
	}
	log.Info("employee update success", zap.String("user_id", payload.UserID))
	return nil
}

func (Eservice *EmployeeService) FindDoctors(log *zap.Logger, search string, organisationID string) ([]dto.Doctor, error) {
	log = ensureLog(log)
	query := `
        SELECT u.*
        FROM users u
        JOIN roles ON roles.id = u.role_id
        WHERE u.organisation_id = $1
          AND roles.name = $2
    `

	args := []interface{}{organisationID, roles.DefaultRoleDoctor}
	idx := 3

	if search != "" {
		query += fmt.Sprintf(`
            AND (u.first_name ILIKE $%d OR u.last_name ILIKE $%d)
        `, idx, idx+1)

		like := "%" + search + "%"
		args = append(args, like, like)
	}
	users, err := Eservice.EmpRepo.ReadDoctors(log, query, args...)
	if err != nil {
		log.Error("doctor list failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, wrapError.ErrDoctorsFetchFailed
	}
	return mapUsersToDoctors(users), nil
}

func mapUsersToDoctors(users []User) []dto.Doctor {
	doctors := make([]dto.Doctor, 0, len(users))
	for _, u := range users {
		doctors = append(doctors, dto.Doctor{
			ID:             u.ID,
			Username:       u.Username,
			FirstName:      u.FirstName,
			LastName:       u.LastName,
			EmailID:        u.EmailID,
			PhoneNumber:    u.PhoneNumber,
			OrganisationID: u.OrganisationID,
			RoleID:         u.RoleID,
			DepartmentID:   u.DepartmentID,
			IsActive:       u.IsActive,
		})
	}
	return doctors
}
func (Eservice *EmployeeService) hashPassword(password string) (hashedPwd []byte, err error) {
	hashedPwd, err = bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		err = errors.New("something went wrong, please contact administrator")
		return
	}
	return
}
func (Eservice *EmployeeService) toEmpModel(passwordHash []byte, payload dto.EmpRequest, roleID string, departmentID string) User {
	username := strings.TrimSpace(payload.UserName)
	if username == "" {
		username = strings.TrimSpace(payload.FirstName + " " + payload.LastName)
	}
	phoneNumber := payload.MobileNumber
	if phoneNumber == "" {
		phoneNumber = payload.PhoneNumber
	}
	return User{
		ID:               uuid.NewString(),
		OrganisationID:   payload.OrganisationID,
		FirstName:        payload.FirstName,
		LastName:         payload.LastName,
		Username:         username,
		EmailID:          payload.EmailID,
		RoleID:           roleID,
		PasswordHash:     string(passwordHash),
		DepartmentID:     departmentID,
		PhoneNumber:      phoneNumber,
		Address:          payload.Address,
		DateOfBirth:      payload.DateOfBirth,
		DateOfJoining:    payload.DateOfJoining,
		ShiftStartTime:   payload.ShiftTimings.StartTime,
		ShiftEndTime:     payload.ShiftTimings.EndTime,
		LicenseNo:        payload.LicenseNo,
		Qualification:    payload.Qualification,
		EmployeeType:     payload.EmployeeType,
		EmergencyEmail:   payload.EmergencyDetails.Email,
		EmergencyName:    payload.EmergencyDetails.Name,
		EmergencyContact: payload.EmergencyDetails.Contact,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func (Eservice *EmployeeService) createEmployeeCode(log *zap.Logger, organisationID string, dateOfJoining string) (string, error) {
	log = ensureLog(log)
	doj := time.Now()
	if dateOfJoining != "" {
		parsed, ok := utils.ParseDate(dateOfJoining)
		if !ok {
			log.Warn("employee code failed",
				zap.String("organisation_id", organisationID),
				zap.String("reason", "invalid_joining_date"),
			)
			return "", wrapError.ErrInvalidRequest
		}
		doj = parsed
	}
	prefix := fmt.Sprintf("%s-%s", constants.EmployeeCodePrefix, doj.Format("20060102"))
	count, err := Eservice.EmpRepo.CountByCodePrefix(log, organisationID, prefix)
	if err != nil {
		log.Error("employee code failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return "", wrapError.ErrEmployeeCreateFailed
	}
	if count == 0 {
		return prefix, nil
	}
	return fmt.Sprintf("%s-%02d", prefix, count+1), nil
}

func (Eservice *EmployeeService) enqueueEmployeeCreated(log *zap.Logger, user User) {
	log = ensureLog(log)
	if Eservice.Notifications == nil {
		log.Warn("employee notification skipped",
			zap.String("user_id", user.ID),
			zap.String("reason", "notifier_missing"),
		)
		return
	}
	org, err := Eservice.OrganisationServicer.GetOrgByID(log, user.OrganisationID)
	if err != nil {
		log.Error("employee notification failed",
			zap.String("user_id", user.ID),
			zap.String("organisation_id", user.OrganisationID),
			zap.String("reason", "org_lookup"),
			zap.Error(err),
		)
		return
	}
	roleName, deptName := Eservice.roleAndDeptNames(log, user.RoleID, user.DepartmentID)
	err = Eservice.Notifications.Create(context.Background(), notificationdto.CreateRequest{
		NotificationType: constants.EmployeeCreatedEvent,
		Subject:          constants.EmployeeCreatedSubject,
		Data: map[string]interface{}{
			"employee_name":   strings.TrimSpace(user.FirstName + " " + user.LastName),
			"employee_email":  user.EmailID,
			"employee_id":     user.ID,
			"role_name":       roleName,
			"department_name": deptName,
			"hospital_name":   org.FacilityName,
			"organisation_id": org.ID,
			"login_url":       Eservice.cfg.LoginUrl,
			"temp_password":   user.TempPassword,
		},
	})
	if err != nil {
		log.Error("employee notification failed",
			zap.String("user_id", user.ID),
			zap.String("organisation_id", user.OrganisationID),
			zap.String("reason", "enqueue"),
			zap.Error(err),
		)
	}
}

func (Eservice *EmployeeService) roleAndDeptNames(log *zap.Logger, roleID string, departmentID string) (string, string) {
	roleName := ""
	if role, err := Eservice.RoleServices.FindByID(log, roleID); err == nil {
		roleName = role.Name
	}
	deptName := ""
	if dept, err := Eservice.DeptServices.FindByID(log, departmentID); err == nil {
		deptName = dept.Name
	}
	return roleName, deptName
}
