package orgbootstrap

import (
	"testing"

	"hospital-backend/central/customers"
	"hospital-backend/internal/department"
	empdto "hospital-backend/internal/employee/dto"
	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/roles"
	"hospital-backend/internal/testutil/servicetest"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubRoles struct {
	rows []roles.Role
}

func (s *stubRoles) FindRoleByOrgID(*zap.Logger, string) ([]roles.Role, error) { return s.rows, nil }
func (s *stubRoles) InsertMany(*zap.Logger, *gorm.DB, string) ([]roles.Role, error) {
	s.rows = []roles.Role{{ID: "role-admin", Name: roles.DefaultRoleHospitalAdmin}}
	return s.rows, nil
}
func (s *stubRoles) FindRoleByNames(*zap.Logger, string, string) (roles.Role, error) {
	return roles.Role{ID: "role-admin", Name: roles.DefaultRoleHospitalAdmin}, nil
}

type stubPerms struct {
	called bool
}

func (s *stubPerms) InsertMany(*zap.Logger, *gorm.DB, []roles.Role, []permissions.Permission, []modules.Modules, string) error {
	s.called = true
	return nil
}

type stubCatalog struct{}

func (stubCatalog) FindMany(*zap.Logger) ([]modules.Modules, []permissions.Permission, error) {
	return []modules.Modules{{ID: "m1", Name: "patient"}}, []permissions.Permission{{ID: "p1", Name: "view"}}, nil
}

type stubDepts struct{}

func (stubDepts) InsertMany(*zap.Logger, *gorm.DB, string) error { return nil }
func (stubDepts) FindDeptByName(*zap.Logger, string, string) (department.Department, error) {
	return department.Department{ID: "dept-admin", Name: department.DefaultDeptAdmin}, nil
}

type stubEmployees struct {
	payload empdto.EmpRequest
}

func (s *stubEmployees) CreateEmployeeInvite(_ *zap.Logger, payload empdto.EmpRequest) (string, string, error) {
	s.payload = payload
	return "user-1", "Temp1!", nil
}

func TestAddFirstUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:orgbootstrap?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE role_permissions (id text, organisation_id text)`).Error; err != nil {
		t.Fatalf("table: %v", err)
	}
	employees := &stubEmployees{}
	perms := &stubPerms{}
	svc := &Service{
		DB:        db,
		Roles:     &stubRoles{},
		RolePerms: perms,
		Catalog:   stubCatalog{},
		Depts:     stubDepts{},
		Employees: employees,
	}
	got, err := svc.AddFirstUser(servicetest.NopLogger(), FirstUserRequest{
		OrganisationID: "org-1",
		FirstName:      "Ada",
		EmailID:        "ada@example.com",
		Username:       "ada",
		DateOfBirth:    "1990-01-02",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !got.PasswordRequired || got.UserID != "user-1" || got.RoleID != "role-admin" || got.TempPassword == "" {
		t.Fatalf("got=%+v", got)
	}
	if employees.payload.RoleID != "role-admin" || employees.payload.DepartmentID != "dept-admin" {
		t.Fatalf("payload=%+v", employees.payload)
	}
	if !perms.called {
		t.Fatal("expected role permissions insert")
	}
}

func TestProvisionHospitalAdminUsesTenantCustomer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:orgbootstrap-admin?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE role_permissions (id text, organisation_id text)`).Error; err != nil {
		t.Fatalf("table: %v", err)
	}
	employees := &stubEmployees{}
	svc := &Service{
		DB:        db,
		Roles:     &stubRoles{},
		RolePerms: &stubPerms{},
		Catalog:   stubCatalog{},
		Depts:     stubDepts{},
		Employees: employees,
		Customers: stubCustomers{customer: customers.Customer{
			ID:        "cust-1",
			FullName:  "Ada Lovelace",
			WorkEmail: "Ada@Example.com",
		}},
	}
	if err := svc.ProvisionHospitalAdmin(servicetest.NopLogger(), "tenant-1", "org-1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if employees.payload.FirstName != "Ada" || employees.payload.LastName != "Lovelace" {
		t.Fatalf("name=%s %s", employees.payload.FirstName, employees.payload.LastName)
	}
	if employees.payload.EmailID != "Ada@Example.com" || employees.payload.UserName != "ada" {
		t.Fatalf("payload=%+v", employees.payload)
	}
	if employees.payload.OrganisationID != "org-1" || employees.payload.RoleID != "role-admin" {
		t.Fatalf("payload=%+v", employees.payload)
	}
}

type stubCustomers struct {
	customer customers.Customer
}

func (s stubCustomers) GetByTenantID(*zap.Logger, string) (customers.Customer, error) {
	return s.customer, nil
}
