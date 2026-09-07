package authentication_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"hospital-backend/config"
	"hospital-backend/internal/authentication"
	"hospital-backend/internal/authentication/dto"
	"hospital-backend/internal/authentication/mocks"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/jwt"
	notificationdto "hospital-backend/internal/notifications/dto"
	rpdto "hospital-backend/internal/rolepermissions/dto"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func newAuthService(t *testing.T, repo authentication.UserRepository, jwtSvc authentication.JwtServicer, rolePerm authentication.RolePermissionServicer) *authentication.UserService {
	t.Helper()
	return authentication.NewService(repo, jwtSvc, rolePerm, servicetest.NoopNotifier{}, &config.Config{
		PasswordResetBaseURL: "http://localhost:5173",
	})
}

func TestServiceUpdatePassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	req := servicetest.ValidUpdatePasswordRequest()
	tokenHash := authentication.HashPasswordResetToken(req.Token)
	user := &employee.User{ID: "user-1"}

	t.Run("validation fail", func(t *testing.T) {
		svc := newAuthService(t, nil, nil, nil)
		err := svc.UpdatePassword(log, dto.UpdatePasswordRequest{
			Token:           req.Token,
			Password:        "short",
			ConfirmPassword: "short",
		})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		svc := newAuthService(t, nil, nil, nil)
		err := svc.UpdatePassword(log, dto.UpdatePasswordRequest{
			Password:        req.Password,
			ConfirmPassword: req.ConfirmPassword,
		})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserByPasswordResetTokenHash(log, authentication.HashPasswordResetToken("wrong-token")).Return(nil, authentication.ErrUserNotFound)
		svc := newAuthService(t, repo, nil, nil)
		bad := req
		bad.Token = "wrong-token"
		err := svc.UpdatePassword(log, bad)
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserByPasswordResetTokenHash(log, tokenHash).Return(nil, errors.New("db error"))
		svc := newAuthService(t, repo, nil, nil)
		err := svc.UpdatePassword(log, req)
		if !errors.Is(err, wrapError.ErrPasswordUpdateFailed) {
			t.Fatalf("expected password update failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserByPasswordResetTokenHash(log, tokenHash).Return(user, nil)
		repo.EXPECT().UpdatePassword(log, "user-1", gomock.Any()).Return(nil)
		svc := newAuthService(t, repo, nil, nil)
		if err := svc.UpdatePassword(log, req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("whitespace trim success", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserByPasswordResetTokenHash(log, tokenHash).Return(user, nil)
		repo.EXPECT().UpdatePassword(log, "user-1", gomock.Any()).Return(nil)
		svc := newAuthService(t, repo, nil, nil)
		reqWithSpace := dto.UpdatePasswordRequest{
			Token:           "  " + req.Token + "  ",
			Password:        "  " + req.Password + "  ",
			ConfirmPassword: "  " + req.ConfirmPassword + "  ",
		}
		if err := svc.UpdatePassword(log, reqWithSpace); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceUpdatePasswordFirstLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	req := dto.FirstLoginPasswordRequest{
		Password:        "newpassword",
		ConfirmPassword: "newpassword",
	}
	user := &employee.User{ID: "user-1"}

	t.Run("validation fail", func(t *testing.T) {
		svc := newAuthService(t, nil, nil, nil)
		err := svc.UpdatePasswordFirstLogin(log, "user-1", dto.FirstLoginPasswordRequest{
			Password:        "short",
			ConfirmPassword: "short",
		})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("missing user id", func(t *testing.T) {
		svc := newAuthService(t, nil, nil, nil)
		err := svc.UpdatePasswordFirstLogin(log, "", req)
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("password already set", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserByID(log, "user-1").Return(&employee.User{ID: "user-1", PasswordHash: "already-hashed"}, nil)
		svc := newAuthService(t, repo, nil, nil)
		err := svc.UpdatePasswordFirstLogin(log, "user-1", req)
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserByID(log, "user-1").Return(user, nil)
		repo.EXPECT().UpdatePassword(log, "user-1", gomock.Any()).Return(nil)
		svc := newAuthService(t, repo, nil, nil)
		if err := svc.UpdatePasswordFirstLogin(log, "user-1", req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceLogout(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	jwtMock := mocks.NewMockJwtServicer(ctrl)

	t.Run("jwt error", func(t *testing.T) {
		jwtMock.EXPECT().LogoutRefreshToken(log, "rt").Return(errors.New("logout error"))
		svc := newAuthService(t, nil, jwtMock, nil)
		if err := svc.Logout(log, "rt"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		jwtMock.EXPECT().LogoutRefreshToken(log, "rt").Return(nil)
		svc := newAuthService(t, nil, jwtMock, nil)
		if err := svc.Logout(log, "rt"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceRefreshToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	jwtMock := mocks.NewMockJwtServicer(ctrl)

	t.Run("session expired", func(t *testing.T) {
		jwtMock.EXPECT().ValidateRefreshToken(log, "rt").Return(jwt.TokenResp{}, wrapError.ErrRefreshSession)
		svc := newAuthService(t, nil, jwtMock, nil)
		_, err := svc.RefreshToken(log, "rt")
		if !errors.Is(err, wrapError.ErrSessionExpired) {
			t.Fatalf("expected session expired, got %v", err)
		}
	})

	t.Run("generic failure", func(t *testing.T) {
		jwtMock.EXPECT().ValidateRefreshToken(log, "rt").Return(jwt.TokenResp{}, errors.New("bad token"))
		svc := newAuthService(t, nil, jwtMock, nil)
		_, err := svc.RefreshToken(log, "rt")
		if !errors.Is(err, wrapError.ErrRefreshFailed) {
			t.Fatalf("expected refresh failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		jwtMock.EXPECT().ValidateRefreshToken(log, "rt").Return(jwt.TokenResp{AccessToken: "a", RefreshToken: "r"}, nil)
		svc := newAuthService(t, nil, jwtMock, nil)
		resp, err := svc.RefreshToken(log, "rt")
		if err != nil || resp.Token != "a" || resp.RefreshToken != "r" {
			t.Fatalf("got %+v err=%v", resp, err)
		}
	})
}

func TestServiceLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := servicetest.NopLogger()
	login := servicetest.ValidLoginUser()

	t.Run("user not found", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserID(log, login.Username).Return(nil, nil)
		svc := newAuthService(t, repo, nil, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials, got %v", err)
		}
	})

	t.Run("db lookup error", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserID(log, login.Username).Return(nil, errors.New("db error"))
		svc := newAuthService(t, repo, nil, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("errUserNotFound", func(t *testing.T) {
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserID(log, login.Username).Return(nil, authentication.ErrUserNotFound)
		svc := newAuthService(t, repo, nil, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials, got %v", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, "correct-pass")
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		svc := newAuthService(t, repo, nil, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials, got %v", err)
		}
	})

	t.Run("clear temp password fail", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		user.TempPassword = login.Password
		user.PasswordHash = ""
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		repo.EXPECT().ClearTempPassword(log, user.ID).Return(errors.New("clear failed"))
		svc := newAuthService(t, repo, nil, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("temp password clears flag", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		user.TempPassword = login.Password
		user.PasswordHash = ""
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)
		roleMock := mocks.NewMockRolePermissionServicer(ctrl)

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		repo.EXPECT().ClearTempPassword(log, user.ID).Return(nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("", nil)
		jwtMock.EXPECT().RefreshToken(user.OrganisationID, user.ID, "").Return(jwt.Claims{
			RefereshToken: "refresh",
			ExpiresAt:     time.Now().Add(time.Hour),
			JTI:           "jti-1",
		}, nil)
		jwtMock.EXPECT().InsertRefreshToken(gomock.Any(), gomock.Any(), user.ID, "jti-1").Return(nil)
		repo.EXPECT().UpdateLastLoginAttempt(log, user.ID, user.LastLoginAttempt+1).Return(nil)
		roleMock.EXPECT().FindModulesByRoleID(user.RoleID).Return(rpdto.RoleAccess{IsAdmin: true}, nil)

		svc := newAuthService(t, repo, jwtMock, roleMock)
		resp, err := svc.Login(log, login)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.PasswordCleared {
			t.Fatal("expected password cleared")
		}
	})

	t.Run("access token fail", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)
		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("", errors.New("token error"))
		svc := newAuthService(t, repo, jwtMock, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("refresh find fail", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)
		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("", errors.New("find error"))
		svc := newAuthService(t, repo, jwtMock, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("update refresh token fail", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("refresh-id", nil)
		jwtMock.EXPECT().UpdateRefreshToken("refresh-id", user.ID, user.OrganisationID).Return("", errors.New("update failed"))

		svc := newAuthService(t, repo, jwtMock, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("mint refresh token fail", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("", nil)
		jwtMock.EXPECT().RefreshToken(user.OrganisationID, user.ID, "").Return(jwt.Claims{}, errors.New("mint failed"))

		svc := newAuthService(t, repo, jwtMock, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("insert refresh token fail", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("", nil)
		jwtMock.EXPECT().RefreshToken(user.OrganisationID, user.ID, "").Return(jwt.Claims{
			RefereshToken: "new-refresh",
			ExpiresAt:     time.Now().Add(time.Hour),
			JTI:           "jti-fail",
		}, nil)
		jwtMock.EXPECT().InsertRefreshToken("new-refresh", gomock.Any(), user.ID, "jti-fail").Return(errors.New("insert failed"))

		svc := newAuthService(t, repo, jwtMock, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("update last login fail", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("refresh-id", nil)
		jwtMock.EXPECT().UpdateRefreshToken("refresh-id", user.ID, user.OrganisationID).Return("refresh", nil)
		repo.EXPECT().UpdateLastLoginAttempt(log, user.ID, user.LastLoginAttempt+1).Return(errors.New("update failed"))

		svc := newAuthService(t, repo, jwtMock, nil)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("update existing refresh", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)
		roleMock := mocks.NewMockRolePermissionServicer(ctrl)

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("refresh-id", nil)
		jwtMock.EXPECT().UpdateRefreshToken("refresh-id", user.ID, user.OrganisationID).Return("updated-refresh", nil)
		repo.EXPECT().UpdateLastLoginAttempt(log, user.ID, user.LastLoginAttempt+1).Return(nil)
		roleMock.EXPECT().FindModulesByRoleID(user.RoleID).Return(rpdto.RoleAccess{}, nil)

		svc := newAuthService(t, repo, jwtMock, roleMock)
		resp, err := svc.Login(log, login)
		if err != nil || resp.RefreshToken != "updated-refresh" {
			t.Fatalf("got %+v err=%v", resp, err)
		}
	})

	t.Run("create new refresh", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("", nil)
		jwtMock.EXPECT().RefreshToken(user.OrganisationID, user.ID, "").Return(jwt.Claims{
			RefereshToken: "new-refresh",
			ExpiresAt:     time.Now().Add(time.Hour),
			JTI:           "jti-2",
		}, nil)
		jwtMock.EXPECT().InsertRefreshToken("new-refresh", gomock.Any(), user.ID, "jti-2").Return(nil)
		repo.EXPECT().UpdateLastLoginAttempt(log, user.ID, user.LastLoginAttempt+1).Return(nil)

		svc := newAuthService(t, repo, jwtMock, nil)
		resp, err := svc.Login(log, login)
		if err != nil || resp.RefreshToken != "new-refresh" {
			t.Fatalf("got %+v err=%v", resp, err)
		}
		if len(resp.Permissions) != 0 {
			t.Fatal("expected empty permissions when role-perm svc nil")
		}
	})

	t.Run("role permissions error", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)
		roleMock := mocks.NewMockRolePermissionServicer(ctrl)

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("refresh-id", nil)
		jwtMock.EXPECT().UpdateRefreshToken("refresh-id", user.ID, user.OrganisationID).Return("refresh", nil)
		repo.EXPECT().UpdateLastLoginAttempt(log, user.ID, user.LastLoginAttempt+1).Return(nil)
		roleMock.EXPECT().FindModulesByRoleID(user.RoleID).Return(rpdto.RoleAccess{}, errors.New("perm error"))

		svc := newAuthService(t, repo, jwtMock, roleMock)
		_, err := svc.Login(log, login)
		if !errors.Is(err, wrapError.ErrLoginFailed) {
			t.Fatalf("expected login failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		user := servicetest.UserWithPassword(t, login.Password)
		repo := mocks.NewMockUserRepository(ctrl)
		jwtMock := mocks.NewMockJwtServicer(ctrl)
		roleMock := mocks.NewMockRolePermissionServicer(ctrl)
		perms := []rpdto.RoleModulePermission{{ModuleName: "patients"}}

		repo.EXPECT().GetUserID(log, login.Username).Return(user, nil)
		jwtMock.EXPECT().AccessToken(user.ID, user.OrganisationID).Return("access-token", nil)
		jwtMock.EXPECT().FindIDByUserID(user.ID).Return("refresh-id", nil)
		jwtMock.EXPECT().UpdateRefreshToken("refresh-id", user.ID, user.OrganisationID).Return("refresh-token", nil)
		repo.EXPECT().UpdateLastLoginAttempt(log, user.ID, user.LastLoginAttempt+1).Return(nil)
		roleMock.EXPECT().FindModulesByRoleID(user.RoleID).Return(rpdto.RoleAccess{IsAdmin: false, Permissions: perms}, nil)

		svc := newAuthService(t, repo, jwtMock, roleMock)
		resp, err := svc.Login(log, login)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Token != "access-token" || resp.UserID != user.ID || len(resp.Permissions) != 1 {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})
}

func TestServiceRequestPasswordReset(t *testing.T) {
	log := servicetest.NopLogger()
	cfg := &config.Config{PasswordResetBaseURL: "http://localhost:5173"}

	t.Run("missing email", func(t *testing.T) {
		svc := authentication.NewService(nil, nil, nil, servicetest.NoopNotifier{}, cfg)
		err := svc.RequestPasswordReset(log, "  ")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("unknown email returns success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockUserRepository(ctrl)
		repo.EXPECT().GetUserID(log, "missing@example.com").Return(nil, authentication.ErrUserNotFound)
		svc := authentication.NewService(repo, nil, nil, servicetest.NoopNotifier{}, cfg)
		if err := svc.RequestPasswordReset(log, "missing@example.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("cooldown", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockUserRepository(ctrl)
		recent := time.Now().UTC().Add(-5 * time.Minute)
		user := &employee.User{ID: "user-1", EmailID: "a@example.com", LastPwdUpdated: &recent}
		repo.EXPECT().GetUserID(log, "a@example.com").Return(user, nil)
		svc := authentication.NewService(repo, nil, nil, servicetest.NoopNotifier{}, cfg)
		err := svc.RequestPasswordReset(log, "a@example.com")
		if !errors.Is(err, wrapError.ErrPasswordResetTooSoon) {
			t.Fatalf("expected too soon, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockUserRepository(ctrl)
		notifier := mocks.NewMockNotificationEnqueuer(ctrl)
		user := &employee.User{
			ID:             "user-1",
			EmailID:        "a@example.com",
			FirstName:      "Ada",
			LastName:       "Lovelace",
			OrganisationID: "org-1",
		}
		var savedHash string
		var plainToken string
		repo.EXPECT().GetUserID(log, "a@example.com").Return(user, nil)
		repo.EXPECT().SavePasswordResetToken(log, "user-1", gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ *zap.Logger, _ string, tokenHash string, _ time.Time) error {
				savedHash = tokenHash
				return nil
			})
		notifier.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req notificationdto.CreateRequest) error {
				data, ok := req.Data.(map[string]interface{})
				if !ok {
					t.Fatal("expected notification data map")
				}
				resetURL, _ := data["reset_url"].(string)
				u, err := url.Parse(resetURL)
				if err != nil {
					t.Fatalf("parse reset url: %v", err)
				}
				plainToken = u.Query().Get("token")
				return nil
			})
		svc := authentication.NewService(repo, nil, nil, notifier, cfg)
		if err := svc.RequestPasswordReset(log, "a@example.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plainToken == "" || savedHash == "" {
			t.Fatal("expected plain token and saved hash")
		}
		if authentication.HashPasswordResetToken(plainToken) != savedHash {
			t.Fatal("saved hash does not match hash of token from reset url")
		}

		repo.EXPECT().GetUserByPasswordResetTokenHash(log, savedHash).Return(&employee.User{ID: "user-1"}, nil)
		repo.EXPECT().UpdatePassword(log, "user-1", gomock.Any()).Return(nil)
		err := svc.UpdatePassword(log, dto.UpdatePasswordRequest{
			Token:           plainToken,
			Password:        "newpassword",
			ConfirmPassword: "newpassword",
		})
		if err != nil {
			t.Fatalf("update password with plain token: %v", err)
		}
	})
}
