package department_test

import (
	"errors"
	"net/http"
	"testing"

	"hospital-backend/internal/department"
	"hospital-backend/internal/department/mocks"
	"hospital-backend/internal/testutil/controllertest"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestFindMany(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockDepartmentServicer)
		wantStatus int
	}{
		{
			name:  "success default page",
			query: "?organisation_id=org-1&limit=10",
			setup: func(m *mocks.MockDepartmentServicer) {
				m.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return([]department.Department{{ID: "dept-1", Name: "Cardiology"}}, int64(1), nil)
			},
			wantStatus: fiber.StatusOK,
		},
		{
			name:  "service error",
			query: "?organisation_id=org-1&limit=10&page=2",
			setup: func(m *mocks.MockDepartmentServicer) {
				m.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("db error"))
			},
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockDepartmentServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			deptCtrl := department.NewDepartmentControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/departments", deptCtrl.FindMany)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/departments" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
