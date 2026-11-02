package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-backend/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type stubRoleIDs struct {
	id  string
	err error
}

func (s stubRoleIDs) FindRoleIDByUserID(*zap.Logger, string) (string, error) {
	return s.id, s.err
}

type stubRoleNames struct {
	name string
	err  error
}

func (s stubRoleNames) FindNameByID(*zap.Logger, string) (string, error) {
	return s.name, s.err
}

func TestRequireRoleHospitalAdminOnly(t *testing.T) {
	if logger.Log == nil {
		logger.Init("development", "error")
	}

	tests := []struct {
		name       string
		userID     string
		lookup     RoleIDFinder
		names      RoleNameFinder
		wantStatus int
	}{
		{
			name:       "hospital admin allowed",
			userID:     "user-admin",
			lookup:     stubRoleIDs{id: "role-ha"},
			names:      stubRoleNames{name: "Hospital Admin"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "super admin denied",
			userID:     "user-super",
			lookup:     stubRoleIDs{id: "role-sa"},
			names:      stubRoleNames{name: "Super Admin"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "doctor denied",
			userID:     "user-doc",
			lookup:     stubRoleIDs{id: "role-doc"},
			names:      stubRoleNames{name: "Doctor"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "missing user denied",
			lookup:     stubRoleIDs{id: "role-ha"},
			names:      stubRoleNames{name: "Hospital Admin"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "role lookup failure denied",
			userID:     "user-1",
			lookup:     stubRoleIDs{err: errors.New("db")},
			names:      stubRoleNames{name: "Hospital Admin"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "role name failure denied",
			userID:     "user-1",
			lookup:     stubRoleIDs{id: "role-1"},
			names:      stubRoleNames{err: errors.New("db")},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/create", func(c *fiber.Ctx) error {
				if tt.userID != "" {
					c.Locals(UserIDKey, tt.userID)
				}
				return c.Next()
			}, RequireRole(tt.lookup, tt.names, "Hospital Admin"), func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/create", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status=%d want=%d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}
