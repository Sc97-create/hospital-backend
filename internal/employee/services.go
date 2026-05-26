package employee

import (
	"errors"
	"fmt"
	"hospital-backend/internal/department"
	"hospital-backend/internal/email"
	"hospital-backend/internal/employee/dto"
	"hospital-backend/internal/employee/utils"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/roles"
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
	//PermServices    *permissions.PermService
}

func NewEmpService(db *gorm.DB, empRepo EmployeeRepository, OrgRepo organisation.OrganisationRepo, roleServices *roles.RoleServices, deptServices *department.DepartmentService) *EmployeeService {
	return &EmployeeService{DB: db, EmpRepo: empRepo, OranisationRepo: OrgRepo, RoleServices: roleServices, DeptServices: deptServices}
}

func (EService *EmployeeService) CreateEmployee(payload dto.EmpRequest) (id string, err error) {
	user := new(User)
	passwordHash, err := EService.hashPassword(payload.Password)
	if err != nil {
		return "", err
	}

	user.ID = uuid.New().String()
	user.OrganisationID = payload.OrganisationID
	user.Username = payload.UserName
	user.EmailID = payload.EmailID
	user.RoleID = payload.RoleID
	user.DepartmentID = payload.DepartmentID
	user.PhoneNumber = payload.PhoneNumber
	user.PasswordHash = string(passwordHash)
	user.IsActive = true
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	err = EService.EmpRepo.Create(user)
	if err != nil {
		return
	}
	organisationData, err := EService.OranisationRepo.GetOrganisationByID(payload.OrganisationID)
	if err != nil {
		return
	}
	err = email.SendNotification(user.Username, organisationData.OrganisationName, "", "", payload.EmailID, "", utils.LoginUrl, utils.AppName)
	if err != nil {
		return
	}
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
func (Eservice *EmployeeService) FindOne(id string) (u *User, err error) {
	u, err = Eservice.EmpRepo.ReadOne(id)
	if err != nil {
		return
	}
	return
}
func (Eservice *EmployeeService) FindMany(limit int, pageno int) (u []User, err error) {
	skip := 0
	if pageno != 0 {
		skip = (pageno - 1) * limit
	}
	u, err = Eservice.EmpRepo.ReadMany(limit, skip)
	if err != nil {
		return
	}
	return
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
	err = Eservice.EmpRepo.Create(&user)
	if err != nil {
		return
	}
	return user.ID, nil
}
func (Eservice *EmployeeService) UpdateAdminProf(payload dto.UpdateRequest) (err error) {
	userData, err := Eservice.FindOne(payload.UserID)
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
	passwordHash, err := Eservice.hashPassword(payload.Password)
	if err != nil {
		return
	}
	if userData.PasswordHash != string(passwordHash) {
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
	return User{
		ID:             uuid.NewString(),
		OrganisationID: payload.OrganisationID,
		FirstName:      payload.FirstName,
		LastName:       payload.LastName,
		Username:       strings.TrimSpace(payload.FirstName + " " + payload.LastName),
		EmailID:        payload.EmailID,
		RoleID:         roleID,
		PasswordHash:   string(passwordHash),
		DepartmentID:   departmentID,
		PhoneNumber:    payload.PhoneNumber,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}
