package rolepermissions_test

import (
	"errors"
	"net/http"
	"testing"

	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/rolepermissions/dto"
	"hospital-backend/internal/rolepermissions/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestFindModulesByRoleID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockRolePermissionServicer)
		wantStatus int
	}{
		{
			name:       "missing role id",
			path:       "/role-permissions/modules",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service error",
			path: "/role-permissions/modules/role-1",
			setup: func(m *mocks.MockRolePermissionServicer) {
				m.EXPECT().FindModulesByRoleID("role-1").Return(dto.RoleAccess{}, wrapError.ErrRolePermissionsFetchFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success via path param",
			path: "/role-permissions/modules/role-1",
			setup: func(m *mocks.MockRolePermissionServicer) {
				m.EXPECT().FindModulesByRoleID("role-1").Return(dto.RoleAccess{
					IsAdmin: false,
					Permissions: []dto.RoleModulePermission{
						{ModuleName: "patient", Permissions: dto.ModulePermissionFlags{View: true}},
					},
				}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
		{
			name: "success via query",
			path: "/role-permissions/modules?role_id=role-2",
			setup: func(m *mocks.MockRolePermissionServicer) {
				m.EXPECT().FindModulesByRoleID("role-2").Return(dto.RoleAccess{IsAdmin: true}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
		{
			name: "generic service error",
			path: "/role-permissions/modules/role-1",
			setup: func(m *mocks.MockRolePermissionServicer) {
				m.EXPECT().FindModulesByRoleID("role-1").Return(dto.RoleAccess{}, errors.New("db error"))
			},
			wantStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockRolePermissionServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			rpCtrl := rolepermissions.NewRolePermissionController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/role-permissions/modules/:roleID?", rpCtrl.FindModulesByRoleID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
