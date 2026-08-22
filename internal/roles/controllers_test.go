package roles_test

import (
	"errors"
	"net/http"
	"testing"

	"hospital-backend/internal/roles"
	"hospital-backend/internal/roles/dto"
	"hospital-backend/internal/roles/mocks"
	"hospital-backend/internal/testutil/controllertest"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestFindMany(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockRoleServicer)
		wantStatus int
	}{
		{
			name:  "success",
			query: "?organisation_id=org-1&limit=10&page=1",
			setup: func(m *mocks.MockRoleServicer) {
				m.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dto.RoleResponse{{ID: "role-1", Name: "Admin"}}, int64(1), nil)
			},
			wantStatus: fiber.StatusOK,
		},
		{
			name:  "service error",
			query: "?organisation_id=org-1&limit=10",
			setup: func(m *mocks.MockRoleServicer) {
				m.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("db error"))
			},
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockRoleServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			roleCtrl := roles.NewRoleControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/roles", roleCtrl.FindMany)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/roles" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
