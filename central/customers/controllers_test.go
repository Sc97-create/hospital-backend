package customers_test

import (
	"net/http"
	"testing"

	"hospital-backend/central/customers"
	dto "hospital-backend/central/customers/dto"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type stubCustomerService struct {
	loginResult dto.LoginResult
	loginErr    error
}

func (s stubCustomerService) Signup(*zap.Logger, dto.SignupPayload) (dto.SignupResult, error) {
	return dto.SignupResult{}, nil
}

func (s stubCustomerService) VerifyEmail(*zap.Logger, string, dto.VerifyEmailPayload) error {
	return nil
}

func (s stubCustomerService) Login(*zap.Logger, dto.LoginPayload) (dto.LoginResult, error) {
	return s.loginResult, s.loginErr
}

func TestControllerLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		svc        stubCustomerService
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing work_email",
			body:       `{"password":"secret12"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"work_email":"ada@acme.com"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "short password",
			body:       `{"work_email":"ada@acme.com","password":"short"}`,
			svc:        stubCustomerService{loginErr: wrapError.ErrInvalidRequest},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid credentials",
			body:       `{"work_email":"ada@acme.com","password":"secret12"}`,
			svc:        stubCustomerService{loginErr: wrapError.ErrInvalidCredentials},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "login failed",
			body:       `{"work_email":"ada@acme.com","password":"secret12"}`,
			svc:        stubCustomerService{loginErr: wrapError.ErrLoginFailed},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			body: `{"work_email":"ada@acme.com","password":"secret12"}`,
			svc: stubCustomerService{loginResult: dto.LoginResult{
				CustomerID:  "cust-1",
				WorkEmail:   "ada@acme.com",
				Status:      "active",
				AccessToken: "tok",
			}},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := customers.NewICustomerController(tt.svc)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/login", ctrl.Login)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/login",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
