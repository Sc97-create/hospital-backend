package employee

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/config"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee/dto"
	"hospital-backend/internal/employee/utils"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/notifications/service"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/roles"
	"hospital-backend/pkg/constants"
	"hospital-backend/pkg/logger"
	wrapError "hospital-backend/shared/error"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type EmployeeService struct {
	DB              *gorm.DB
	EmpRepo         EmployeeRepository
	OranisationRepo organisation.OrganisationRepo
	RoleServices    *roles.RoleServices
	DeptServices    *department.DepartmentService
	Notifications   *service.Notificationservice
	cfg             *config.Config
}

func NewEmpService(db *gorm.DB, empRepo EmployeeRepository, OrgRepo organisation.OrganisationRepo, roleServices *roles.RoleServices, deptServices *department.DepartmentService, cfg *config.Config) *EmployeeService {
	return &EmployeeService{DB: db, EmpRepo: empRepo, OranisationRepo: OrgRepo, RoleServices: roleServices, DeptServices: deptServices, cfg: cfg}
}

func (EService *EmployeeService) CreateEmployee(payload dto.EmpRequest) (id string, err error) {
	tempPassword := utils.CreateTempPassword(payload.FirstName, payload.DateOfBirth)
	// passwordHash, err := EService.hashPassword(tempPassword)
	// if err != nil {
	// 	return "", err
	// }

	user := EService.toEmpModel([]byte{}, payload, payload.RoleID, payload.DepartmentID)
	user.TempPassword = tempPassword
	code, err := EService.createEmployeeCode(payload.OrganisationID, payload.DateOfJoining)
	if err != nil {
		return "", err
	}
	user.EmployeeCode = code
	err = EService.EmpRepo.Create(&user)
	if err != nil {
		return
	}
	EService.enqueueEmployeeCreated(user)
	return user.ID, nil
}

func (Eservice *EmployeeService) DeleteEmployee(userID string) (err error) {
	if userID == "" {
		err = errors.New("userid is not passed")
		return
	}
	err = Eservice.EmpRepo.DeleteOne(userID)
	if err != nil {
		return
	}
	return
}
func (Eservice *EmployeeService) FindOne(id string) (dto.EmployeeResponse, error) {
	row, err := Eservice.EmpRepo.ReadOne(id)
	if err != nil {
		return dto.EmployeeResponse{}, err
	}
	return Eservice.mapToEmployeeResponse(*row), nil
}
func (Eservice *EmployeeService) FindMany(req dto.FindManyRequest) (employeeResp []dto.EmployeeResponse, total int64, err error) {
	req.Search = strings.TrimSpace(req.Search)
	limitInt, skip := Eservice.getPageSkip(req.Limit, req.PageNo)
	users, err := Eservice.EmpRepo.ReadMany(limitInt, skip, req.OrganisationID, req.Search)
	if err != nil {
		return
	}
	total, err = Eservice.EmpRepo.Count(req.OrganisationID, req.Search)
	if err != nil {
		return
	}
	employeeResp = Eservice.arrayMapToEmployeeResponse(users)
	return
}

func (Eservice *EmployeeService) getPageSkip(limit int, pageNo int) (int, int) {
	skip := 0
	if pageNo != 0 {
		skip = (pageNo - 1) * limit
	}
	return limit, skip
}

func (Eservice *EmployeeService) arrayMapToEmployeeResponse(rows []employeeListRow) []dto.EmployeeResponse {
	employeeResponse := []dto.EmployeeResponse{}
	for _, each := range rows {
		employeeResponse = append(employeeResponse, Eservice.mapToEmployeeResponse(each))
	}
	return employeeResponse
}

func (Eservice *EmployeeService) mapToEmployeeResponse(row employeeListRow) dto.EmployeeResponse {
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

func (Eservice *EmployeeService) FindRoleIDByUserID(userID string) (string, error) {
	return Eservice.EmpRepo.FindRoleIDByUserID(userID)
}
func (Eservice *EmployeeService) CreateAdminProf(payload dto.EmpRequest) (userID string, err error) {
	passwordHash, err := Eservice.hashPassword(payload.Password)
	if err != nil {
		return
	}
	role, err := Eservice.RoleServices.FindRoleByNames(payload.OrganisationID, roles.DefaultRoleAdmin)
	if err != nil {
		return
	}
	department, err := Eservice.DeptServices.FindDeptByName(payload.OrganisationID, department.DefaultDeptAdmin)
	if err != nil {
		return
	}
	user := Eservice.toEmpModel(passwordHash, payload, role.ID, department.ID)
	code, err := Eservice.createEmployeeCode(payload.OrganisationID, payload.DateOfJoining)
	if err != nil {
		return
	}
	user.EmployeeCode = code
	err = Eservice.EmpRepo.Create(&user)
	if err != nil {
		return
	}
	return user.ID, nil
}
func (Eservice *EmployeeService) UpdateAdminProf(payload dto.UpdateRequest) (err error) {
	userData, err := Eservice.EmpRepo.ReadOne(payload.UserID)
	if err != nil {
		return
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
			return hashErr
		}
		updateUser["password_hash"] = string(passwordHash)
	}
	if payload.FirstName != "" && payload.LastName != "" {
		updateUser["username"] = payload.FirstName + " " + payload.LastName
	}
	return Eservice.EmpRepo.Update(payload.UserID, updateUser)
}

func (Eservice *EmployeeService) FindDoctors(search string, organisationID string) (u []User, err error) {
	query := `
        SELECT u.*
        FROM users u
        JOIN roles ON roles.id = u.role_id
        WHERE u.organisation_id = $1
    `

	args := []interface{}{organisationID}
	idx := 2

	if search != "" {
		query += fmt.Sprintf(`
            AND (u.first_name ILIKE $%d OR u.last_name ILIKE $%d)
        `, idx, idx+1)

		like := "%" + search + "%"
		args = append(args, like, like)
		idx += 2
	}
	u, err = Eservice.EmpRepo.ReadDoctors(query, args...)
	if err != nil {
		return
	}
	return
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

func (Eservice *EmployeeService) createEmployeeCode(organisationID string, dateOfJoining string) (string, error) {
	doj := time.Now()
	if dateOfJoining != "" {
		parsed, ok := utils.ParseDate(dateOfJoining)
		if !ok {
			return "", wrapError.ErrInvalidRequest
		}
		doj = parsed
	}
	prefix := fmt.Sprintf("%s-%s", constants.EmployeeCodePrefix, doj.Format("20060102"))
	count, err := Eservice.EmpRepo.CountByCodePrefix(organisationID, prefix)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return prefix, nil
	}
	return fmt.Sprintf("%s-%02d", prefix, count+1), nil
}

func (Eservice *EmployeeService) enqueueEmployeeCreated(user User) {
	if Eservice.Notifications == nil {
		return
	}
	org, err := Eservice.OranisationRepo.GetOrganisationByID(logger.Log, user.OrganisationID)
	if err != nil {
		return
	}
	roleName, deptName := Eservice.roleAndDeptNames(user.RoleID, user.DepartmentID)
	_ = Eservice.Notifications.Create(context.Background(), notificationdto.CreateRequest{
		NotificationType: constants.EmployeeCreatedEvent,
		Subject:          constants.EmployeeCreatedSubject,
		Data: map[string]interface{}{
			"employee_name":   strings.TrimSpace(user.FirstName + " " + user.LastName),
			"employee_email":  user.EmailID,
			"employee_id":     user.ID,
			"role_name":       roleName,
			"department_name": deptName,
			"hospital_name":   org.OrganisationName,
			"organisation_id": org.ID,
			"login_url":       Eservice.cfg.LoginUrl,
			"temp_password":   user.TempPassword,
		},
	})
}

func (Eservice *EmployeeService) roleAndDeptNames(roleID string, departmentID string) (string, string) {
	roleName := ""
	if role, err := Eservice.RoleServices.FindByID(roleID); err == nil {
		roleName = role.Name
	}
	deptName := ""
	if dept, err := Eservice.DeptServices.FindByID(departmentID); err == nil {
		deptName = dept.Name
	}
	return roleName, deptName
}
