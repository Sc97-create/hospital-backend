package license_test

import (
	"net/http"
	"testing"

	"hospital-backend/internal/license"
	"hospital-backend/internal/license/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestVerifyLicense(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		setup      func(*mocks.MockLicenseServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			path:       "/license/org-1/verify",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing license key",
			path:       "/license/org-1/verify",
			body:       `{}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "invalid license",
			path: "/license/org-1/verify",
			body: `{"license_key":"bad-key"}`,
			setup: func(m *mocks.MockLicenseServicer) {
				m.EXPECT().VerifyLicense(gomock.Any(), gomock.Any(), gomock.Any()).Return(wrapError.ErrLicenseInvalid)
			},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name: "success",
			path: "/license/org-1/verify",
			body: `{"license_key":"valid-key"}`,
			setup: func(m *mocks.MockLicenseServicer) {
				m.EXPECT().VerifyLicense(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockLicenseServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/license/:organisationID/verify", func(c *fiber.Ctx) error {
					return license.VerifyLicense(c, mock)
				})
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   tt.path,
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
