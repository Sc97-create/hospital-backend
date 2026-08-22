package employee_test

import (
	"errors"
	"net/http"
	"testing"

	"hospital-backend/internal/employee"
	"hospital-backend/internal/employee/dto"
	"hospital-backend/internal/employee/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

const validEmployeeBody = `{
	"organisation_id":"org-1",
	"first_name":"Jane",
	"last_name":"Doe",
	"mobile_number":"9876543210",
	"email_id":"jane@example.com",
	"address":"123 Main St",
	"date_of_birth":"1990-01-01",
	"date_of_joining":"2020-01-01",
	"role_id":"role-1",
	"dept_id":"dept-1",
	"license_no":"LIC-1",
	"qualification":"MBBS",
	"employee_type":"full_time",
	"shift_timings":{"start_time":"09:00","end_time":"17:00"},
	"emergency_details":{"email":"em@example.com","name":"Emergency","contact":"9876543211"}
}`

func TestAdd(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockEmployeeServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: 409,
		},
		{
			name:       "missing organisation_id",
			body:       `{"first_name":"Jane"}`,
			wantStatus: 409,
		},
		{
			name: "success",
			body: validEmployeeBody,
			setup: func(m *mocks.MockEmployeeServicer) {
				m.EXPECT().CreateEmployee(gomock.Any()).Return("emp-1", nil)
			},
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockEmployeeServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			empCtrl := employee.NewEmployeeControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/employees", empCtrl.Add)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/employees",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockEmployeeServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: 409,
		},
		{
			name:       "missing user_id",
			body:       `{}`,
			wantStatus: 409,
		},
		{
			name: "success",
			body: `{"user_id":"user-1"}`,
			setup: func(m *mocks.MockEmployeeServicer) {
				m.EXPECT().DeleteEmployee(gomock.Any()).Return(nil)
			},
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockEmployeeServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			empCtrl := employee.NewEmployeeControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/employees/delete", empCtrl.Delete)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/employees/delete",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindByID(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockEmployeeServicer)
		wantStatus int
	}{
		{
			name:  "success",
			query: "?user_id=user-1",
			setup: func(m *mocks.MockEmployeeServicer) {
				m.EXPECT().FindOne(gomock.Any()).Return(dto.EmployeeResponse{EmployeeID: "user-1"}, nil)
			},
			wantStatus: 200,
		},
		{
			name:  "service error",
			query: "?user_id=user-1",
			setup: func(m *mocks.MockEmployeeServicer) {
				m.EXPECT().FindOne(gomock.Any()).Return(dto.EmployeeResponse{}, wrapError.ErrInvalidRequest)
			},
			wantStatus: 409,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockEmployeeServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			empCtrl := employee.NewEmployeeControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/employees", empCtrl.FindByID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/employees" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindMany(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockEmployeeServicer(ctrl)
	mock.EXPECT().FindMany(gomock.Any()).Return([]dto.EmployeeResponse{{EmployeeID: "user-1"}}, int64(1), nil)
	empCtrl := employee.NewEmployeeControllerInterface(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/employees/list", empCtrl.FindMany)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/employees/list?organisation_id=org-1&limit=10&page_no=1",
	})
	controllertest.AssertStatus(t, resp, 200)
}

func TestCreateAdmin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockEmployeeServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: 409,
		},
		{
			name: "success",
			body: `{"organisation_id":"org-1","first_name":"Admin","last_name":"User","password":"secret","confirm_password":"secret","email_id":"admin@example.com","mob_no":"9876543210"}`,
			setup: func(m *mocks.MockEmployeeServicer) {
				m.EXPECT().CreateAdminProf(gomock.Any()).Return("admin-1", nil)
			},
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockEmployeeServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			empCtrl := employee.NewEmployeeControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/employees/admin", empCtrl.CreateAdmin)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/employees/admin",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockEmployeeServicer)
		wantStatus int
	}{
		{
			name:       "password mismatch",
			body:       `{"user_id":"u1","password":"a","confirm_password":"b"}`,
			wantStatus: 409,
		},
		{
			name: "success",
			body: `{"user_id":"u1","password":"secret","confirm_password":"secret"}`,
			setup: func(m *mocks.MockEmployeeServicer) {
				m.EXPECT().UpdateAdminProf(gomock.Any()).Return(nil)
			},
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockEmployeeServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			empCtrl := employee.NewEmployeeControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/employees/update", empCtrl.UpdateUser)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/employees/update",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindDoctors(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockEmployeeServicer(ctrl)
	mock.EXPECT().FindDoctors(gomock.Any(), gomock.Any()).Return([]dto.Doctor{{ID: "doc-1"}}, nil)
	empCtrl := employee.NewEmployeeControllerInterface(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/employees/doctors", empCtrl.FindDoctors)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/employees/doctors?name=john&organisation_id=org-1",
	})
	controllertest.AssertStatus(t, resp, 200)
}

func TestFindManyServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockEmployeeServicer(ctrl)
	mock.EXPECT().FindMany(gomock.Any()).Return(nil, int64(0), errors.New("db error"))
	empCtrl := employee.NewEmployeeControllerInterface(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/employees/list", empCtrl.FindMany)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/employees/list?organisation_id=org-1",
	})
	controllertest.AssertStatus(t, resp, 409)
}
