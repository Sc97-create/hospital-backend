package permissions_test

import (
	"errors"
	"net/http"
	"testing"

	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/permissions/mocks"
	"hospital-backend/internal/testutil/controllertest"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestFindMany(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*mocks.MockPermissionServicer)
		wantStatus int
	}{
		{
			name: "success",
			setup: func(m *mocks.MockPermissionServicer) {
				m.EXPECT().FindMany().Return([]modules.Modules{{ID: "mod-1"}}, []permissions.Permission{{ID: "perm-1"}}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
		{
			name: "service error",
			setup: func(m *mocks.MockPermissionServicer) {
				m.EXPECT().FindMany().Return(nil, nil, errors.New("db error"))
			},
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockPermissionServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/permissions", func(c *fiber.Ctx) error {
					return permissions.FindMany(c, mock)
				})
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/permissions",
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
