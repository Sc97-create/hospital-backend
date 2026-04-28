package employee

import (
	"errors"
	"hospital-backend/internal/department"
	deptUtil "hospital-backend/internal/department/utils"
	"hospital-backend/internal/email"
	"hospital-backend/internal/employee/dto"
	"hospital-backend/internal/employee/utils"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/rolepermissions"
	rolePermissionUtil "hospital-backend/internal/rolepermissions/utils"
	"hospital-backend/internal/roles"
	roleUtil "hospital-backend/internal/roles/utils"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type EmployeeService struct {
	DB               *gorm.DB
	EmpRepo          EmployeeRepository
	OranisationRepo  organisation.OrganisationRepo
	RoleDB           *roles.RoleDB
	DeptDB           *department.DepartmentDB
	PermDB           *permissions.PermissionDB
	RolePermissionDB *rolepermissions.RolePermissionDb
}

func NewEmpService(db *gorm.DB, empRepo EmployeeRepository, OrgRepo organisation.OrganisationRepo, roleRepo *roles.RoleDB, deptDB *department.DepartmentDB, rolePermissionDB *rolepermissions.RolePermissionDb, permDB *permissions.PermissionDB) *EmployeeService {
	return &EmployeeService{DB: db, EmpRepo: empRepo, OranisationRepo: OrgRepo, RoleDB: roleRepo, DeptDB: deptDB, RolePermissionDB: rolePermissionDB, PermDB: permDB}
}

func (EService *EmployeeService) CreateEmployee(payload dto.EmpRequest) (id string, err error) {
	user := new(User)
	passwordHash, err := EService.hashPassword(payload.Password)
	if err != nil {
		return "", err
	}
	err = EService.DB.Transaction(func(tx *gorm.DB) error {
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
		err = EService.EmpRepo.Create(tx, user)
		if err != nil {
			return err
		}
		return nil
	})
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
	user := new(User)
	passwordHash, err := Eservice.hashPassword(payload.Password)
	if err != nil {
		return
	}
	user.ID = uuid.New().String()
	roleArr := roleUtil.AddDefaultRoles(user.ID)
	deptArr := deptUtil.AddDefaultDepartment(user.ID)
	permArr, err := Eservice.PermDB.GetPermissionByName()
	if err != nil {
		return
	}
	rolePermissionArr := rolePermissionUtil.AddDefaultRolePermissions(roleArr[0].ID, permArr)
	err = Eservice.DB.Transaction(func(tx *gorm.DB) error {
		err = Eservice.RoleDB.BatchInsert(tx, roleArr, 2)
		if err != nil {
			return err
		}
		err = Eservice.DeptDB.BatchInsert(tx, deptArr, 2)
		if err != nil {
			return err
		}
		err = Eservice.RolePermissionDB.BatchCreate(tx, rolePermissionArr, 2)
		if err != nil {
			return err
		}
		user.OrganisationID = payload.OrganisationID
		user.FirstName = payload.FirstName
		user.LastName = payload.LastName
		user.Username = strings.TrimSpace(user.FirstName + " " + user.LastName)
		user.EmailID = payload.EmailID
		user.RoleID = roleArr[0].ID
		user.PasswordHash = string(passwordHash)
		user.DepartmentID = deptArr[0].ID
		user.PhoneNumber = payload.PhoneNumber
		user.OrganisationID = payload.OrganisationID
		user.IsActive = true
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		err = Eservice.EmpRepo.Create(tx, user)
		if err != nil {
			return err
		}
		return nil
	})
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

func (Eservice *EmployeeService) FindDoctors(name string) (u []User, err error) {
	u, err = Eservice.EmpRepo.ReadDoctors(name)
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
