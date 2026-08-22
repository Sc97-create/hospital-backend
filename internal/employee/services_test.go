package employee_test

import (
	"errors"
	"testing"
	"time"

	"hospital-backend/internal/employee"
	"hospital-backend/internal/employee/mocks"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
)

func newEmployeeService(t *testing.T, repo employee.EmployeeRepository) *employee.EmployeeService {
	return employee.NewEmpService(nil, repo, nil, nil, nil, servicetest.NoopNotifier{}, nil)
}

func TestServiceDeleteEmployee(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("empty id", func(t *testing.T) {
		svc := newEmployeeService(t, nil)
		if err := svc.DeleteEmployee(""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := mocks.NewMockEmployeeRepository(ctrl)
		repo.EXPECT().DeleteOne("user-1").Return(errors.New("delete error"))
		svc := newEmployeeService(t, repo)
		if err := svc.DeleteEmployee("user-1"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockEmployeeRepository(ctrl)
		repo.EXPECT().DeleteOne("user-1").Return(nil)
		svc := newEmployeeService(t, repo)
		if err := svc.DeleteEmployee("user-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceFindOne(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockEmployeeRepository(ctrl)

	t.Run("repo error", func(t *testing.T) {
		repo.EXPECT().ReadOne("e1").Return(nil, errors.New("not found"))
		svc := newEmployeeService(t, repo)
		if _, err := svc.FindOne("e1"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().ReadOne("e1").Return(&employee.EmployeeListRow{
			ID:        "e1",
			FirstName: "Jane",
			LastName:  "Doe",
			IsActive:  true,
		}, nil)
		svc := newEmployeeService(t, repo)
		resp, err := svc.FindOne("e1")
		if err != nil || resp.EmployeeID != "e1" || resp.EmployeeName != "Jane Doe" {
			t.Fatalf("got %+v err=%v", resp, err)
		}
	})
}

func TestServiceFindMany(t *testing.T) {
	ctrl := gomock.NewController(t)
	req := servicetest.ValidEmpFindManyRequest()

	t.Run("read error", func(t *testing.T) {
		repo := mocks.NewMockEmployeeRepository(ctrl)
		repo.EXPECT().ReadMany(10, 0, req.OrganisationID, "").Return(nil, errors.New("read error"))
		svc := newEmployeeService(t, repo)
		if _, _, err := svc.FindMany(req); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("count error", func(t *testing.T) {
		repo := mocks.NewMockEmployeeRepository(ctrl)
		repo.EXPECT().ReadMany(10, 0, req.OrganisationID, "").Return([]employee.EmployeeListRow{{ID: "e1"}}, nil)
		repo.EXPECT().Count(req.OrganisationID, "").Return(int64(0), errors.New("count error"))
		svc := newEmployeeService(t, repo)
		if _, _, err := svc.FindMany(req); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockEmployeeRepository(ctrl)
		repo.EXPECT().ReadMany(10, 0, req.OrganisationID, "").Return([]employee.EmployeeListRow{
			{ID: "e1", FirstName: "Jane", LastName: "Doe", IsActive: true},
		}, nil)
		repo.EXPECT().Count(req.OrganisationID, "").Return(int64(1), nil)
		svc := newEmployeeService(t, repo)
		resp, total, err := svc.FindMany(req)
		if err != nil || total != 1 || len(resp) != 1 {
			t.Fatalf("got len=%d total=%d err=%v", len(resp), total, err)
		}
	})
}

func TestServiceFindRoleIDByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockEmployeeRepository(ctrl)

	t.Run("repo error", func(t *testing.T) {
		repo.EXPECT().FindRoleIDByUserID("user-1").Return("", errors.New("not found"))
		svc := newEmployeeService(t, repo)
		if _, err := svc.FindRoleIDByUserID("user-1"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().FindRoleIDByUserID("user-1").Return("role-1", nil)
		svc := newEmployeeService(t, repo)
		roleID, err := svc.FindRoleIDByUserID("user-1")
		if err != nil || roleID != "role-1" {
			t.Fatalf("got %q err=%v", roleID, err)
		}
	})
}

func TestServiceFindDoctors(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockEmployeeRepository(ctrl)

	t.Run("repo error", func(t *testing.T) {
		repo.EXPECT().ReadDoctors(gomock.Any(), "org-1").Return(nil, errors.New("read error"))
		svc := newEmployeeService(t, repo)
		if _, err := svc.FindDoctors("", "org-1"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().ReadDoctors(gomock.Any(), "org-1").Return([]employee.User{{ID: "doc-1"}}, nil)
		svc := newEmployeeService(t, repo)
		got, err := svc.FindDoctors("", "org-1")
		if err != nil || len(got) != 1 || got[0].ID != "doc-1" {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestGetPageSkip(t *testing.T) {
	svc := &employee.EmployeeService{}

	limit, skip := svc.TestGetPageSkip(10, 0)
	if limit != 10 || skip != 0 {
		t.Fatalf("page 0: got limit=%d skip=%d", limit, skip)
	}

	limit, skip = svc.TestGetPageSkip(10, 2)
	if limit != 10 || skip != 10 {
		t.Fatalf("page 2: got limit=%d skip=%d", limit, skip)
	}
}

func TestMapToEmployeeResponse(t *testing.T) {
	svc := &employee.EmployeeService{}

	t.Run("username fallback and active", func(t *testing.T) {
		row := employee.EmployeeListRow{
			ID:             "e1",
			FirstName:      "Jane",
			LastName:       "Doe",
			IsActive:       true,
			OrganisationID: "org-1",
		}
		resp := svc.TestMapToEmployeeResponse(row)
		if resp.EmployeeName != "Jane Doe" {
			t.Fatalf("expected name fallback, got %q", resp.EmployeeName)
		}
		if resp.EmployeeStatus != "active" {
			t.Fatalf("expected active, got %q", resp.EmployeeStatus)
		}
	})

	t.Run("uses username when set", func(t *testing.T) {
		row := employee.EmployeeListRow{Username: "jdoe", IsActive: false}
		resp := svc.TestMapToEmployeeResponse(row)
		if resp.EmployeeName != "jdoe" {
			t.Fatalf("expected jdoe, got %q", resp.EmployeeName)
		}
		if resp.EmployeeStatus != "inactive" {
			t.Fatalf("expected inactive, got %q", resp.EmployeeStatus)
		}
	})
}

func TestCreateEmployeeCode(t *testing.T) {
	doj := "2026-03-15"
	parsed, _ := time.Parse("2006-01-02", doj)
	prefix := constants.EmployeeCodePrefix + "-" + parsed.Format("20060102")

	t.Run("invalid DOJ", func(t *testing.T) {
		svc := &employee.EmployeeService{}
		_, err := svc.TestCreateEmployeeCode("org-1", "not-a-date")
		if err != wrapError.ErrInvalidRequest {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := employee.NewEmployeeServiceForTest(codeCountRepo{err: errors.New("count error")})
		_, err := svc.TestCreateEmployeeCode("org-1", doj)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("first code", func(t *testing.T) {
		svc := employee.NewEmployeeServiceForTest(codeCountRepo{count: 0})
		code, err := svc.TestCreateEmployeeCode("org-1", doj)
		if err != nil || code != prefix {
			t.Fatalf("got %q err=%v", code, err)
		}
	})

	t.Run("count-based suffix", func(t *testing.T) {
		svc := employee.NewEmployeeServiceForTest(codeCountRepo{count: 2})
		code, err := svc.TestCreateEmployeeCode("org-1", doj)
		if err != nil || code != prefix+"-03" {
			t.Fatalf("got %q err=%v", code, err)
		}
	})
}

type codeCountRepo struct {
	count int64
	err   error
}

func (codeCountRepo) Create(*employee.User) error { panic("unused") }
func (codeCountRepo) Update(string, map[string]interface{}) error {
	panic("unused")
}
func (codeCountRepo) DeleteOne(string) error { panic("unused") }
func (codeCountRepo) ReadMany(int, int, string, string) ([]employee.EmployeeListRow, error) {
	panic("unused")
}
func (codeCountRepo) ReadOne(string) (*employee.EmployeeListRow, error) { panic("unused") }
func (codeCountRepo) ReadDoctors(string, ...any) ([]employee.User, error) {
	panic("unused")
}
func (codeCountRepo) Count(string, string) (int64, error) { panic("unused") }
func (r codeCountRepo) CountByCodePrefix(string, string) (int64, error) {
	return r.count, r.err
}
func (codeCountRepo) FindRoleIDByUserID(string) (string, error) { panic("unused") }
