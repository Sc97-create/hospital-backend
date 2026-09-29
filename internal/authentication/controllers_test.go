package authentication_test

import (
	"errors"
	"net/http"
	"testing"

	"hospital-backend/internal/authentication"
	"hospital-backend/internal/authentication/dto"
	"hospital-backend/internal/authentication/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockAuthServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing user_name",
			body:       `{"password":"secret"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"user_name":"admin"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "invalid credentials",
			body: `{"user_name":"admin","password":"wrong"}`,
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().Login(gomock.Any(), gomock.Any()).Return(dto.LoginResponse{}, wrapError.ErrInvalidCredentials)
			},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name: "success",
			body: `{"user_name":"admin","password":"secret"}`,
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().Login(gomock.Any(), gomock.Any()).Return(dto.LoginResponse{Token: "tok", RefreshToken: "ref", Message: "ok"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAuthServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			authCtrl := authentication.NewAuthController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/login", authCtrl.Login)
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

func TestRefresh(t *testing.T) {
	tests := []struct {
		name       string
		cookies    map[string]string
		setup      func(*mocks.MockAuthServicer)
		wantStatus int
	}{
		{
			name:       "missing cookie",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:    "session expired",
			cookies: map[string]string{"refresh_token": "expired"},
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().RefreshToken(gomock.Any(), gomock.Any()).Return(dto.LoginResponse{}, wrapError.ErrSessionExpired)
			},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:    "success",
			cookies: map[string]string{"refresh_token": "valid"},
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().RefreshToken(gomock.Any(), gomock.Any()).Return(dto.LoginResponse{Token: "new", RefreshToken: "new-ref"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAuthServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			authCtrl := authentication.NewAuthController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/refresh", authCtrl.Refresh)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method:  http.MethodPost,
				Path:    "/refresh",
				Cookies: tt.cookies,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestLogout(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAuthServicer(ctrl)
	mock.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(nil)
	authCtrl := authentication.NewAuthController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/logout", authCtrl.Logout)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/logout",
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
}

func TestUpdatePassword(t *testing.T) {
	validBody := `{"token":"reset-token","password":"secret12","confirm_password":"secret12"}`
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockAuthServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing token",
			body:       `{"password":"secret12","confirm_password":"secret12"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"token":"t","confirm_password":"a"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service invalid request",
			body: validBody,
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Return(wrapError.ErrInvalidRequest)
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: validBody,
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAuthServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			authCtrl := authentication.NewAuthController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/password", authCtrl.UpdatePassword)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/password",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestUpdatePasswordFirstLogin(t *testing.T) {
	validBody := `{"password":"secret12","confirm_password":"secret12"}`
	tests := []struct {
		name       string
		userID     string
		body       string
		setup      func(*mocks.MockAuthServicer)
		wantStatus int
	}{
		{
			name:       "missing user id",
			body:       validBody,
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "invalid json",
			userID:     "user-1",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing password",
			userID:     "user-1",
			body:       `{"confirm_password":"secret12"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:   "service invalid request",
			userID: "user-1",
			body:   validBody,
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().UpdatePasswordFirstLogin(gomock.Any(), "user-1", gomock.Any()).Return(wrapError.ErrInvalidRequest)
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:   "success",
			userID: "user-1",
			body:   validBody,
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().UpdatePasswordFirstLogin(gomock.Any(), "user-1", gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAuthServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			authCtrl := authentication.NewAuthController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				if tt.userID != "" {
					app.Use(controllertest.WithUserID(tt.userID))
				}
				app.Post("/password-first-login", authCtrl.UpdatePasswordFirstLogin)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/password-first-login",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestLoginInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAuthServicer(ctrl)
	mock.EXPECT().Login(gomock.Any(), gomock.Any()).Return(dto.LoginResponse{}, errors.New("db down"))
	authCtrl := authentication.NewAuthController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/login", authCtrl.Login)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/login",
		Body:   `{"user_name":"admin","password":"secret"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusInternalServerError)
}

func TestRequestPasswordReset(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockAuthServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing email",
			body:       `{"password":"x"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "cooldown",
			body: `{"email_id":"a@example.com"}`,
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().RequestPasswordReset(gomock.Any(), "a@example.com").Return(wrapError.ErrPasswordResetTooSoon)
			},
			wantStatus: fiber.StatusTooManyRequests,
		},
		{
			name: "success",
			body: `{"email_id":"a@example.com"}`,
			setup: func(m *mocks.MockAuthServicer) {
				m.EXPECT().RequestPasswordReset(gomock.Any(), "a@example.com").Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAuthServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			authCtrl := authentication.NewAuthController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/requestPasswordReset", authCtrl.RequestPasswordReset)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/requestPasswordReset",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
