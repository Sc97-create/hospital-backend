package authentication

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hospital-backend/config"
	"hospital-backend/internal/authentication/dto"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/jwt"
	notificationdto "hospital-backend/internal/notifications/dto"
	rpdto "hospital-backend/internal/rolepermissions/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost      = 8
	minPasswordLen  = 8
	resetTokenBytes = 32
)

type JwtServicer interface {
	AccessToken(userID, organisationID string) (string, error)
	FindIDByUserID(userID string) (string, error)
	UpdateRefreshToken(refreshID, userID, organisationID string) (string, error)
	RefreshToken(organisationID, userID, refreshID string) (jwt.Claims, error)
	InsertRefreshToken(token string, expiry time.Time, userID, refreshID string) error
	ValidateRefreshToken(log *zap.Logger, token string) (jwt.TokenResp, error)
	LogoutRefreshToken(log *zap.Logger, refreshToken string) error
}

type RolePermissionServicer interface {
	FindModulesByRoleID(roleID string) (rpdto.RoleAccess, error)
}

type NotificationEnqueuer interface {
	Create(ctx context.Context, data notificationdto.CreateRequest) error
}

type UserService struct {
	Repo              UserRepository
	JwtService        JwtServicer
	RolePermissionSvc RolePermissionServicer
	Notifications     NotificationEnqueuer
	cfg               *config.Config
}

func NewService(repo UserRepository, jwtService JwtServicer, rolePermSvc RolePermissionServicer, notifications NotificationEnqueuer, cfg *config.Config) *UserService {
	return &UserService{
		Repo:              repo,
		JwtService:        jwtService,
		RolePermissionSvc: rolePermSvc,
		Notifications:     notifications,
		cfg:               cfg,
	}
}

func (a *UserService) Login(log *zap.Logger, L dto.LoginUser) (dto.LoginResponse, error) {
	log = ensureLog(log)
	user, err := a.Repo.GetUserID(log, L.Username)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			log.Warn("login failed",
				zap.String("username", L.Username),
				zap.String("reason", "user_not_found"),
			)
			return dto.LoginResponse{}, wrapError.ErrInvalidCredentials
		}
		log.Error("login failed",
			zap.String("username", L.Username),
			zap.String("reason", "user_lookup"),
			zap.Error(err),
		)
		return dto.LoginResponse{}, wrapError.ErrLoginFailed
	}
	if user == nil {
		log.Warn("login failed",
			zap.String("username", L.Username),
			zap.String("reason", "user_not_found"),
		)
		return dto.LoginResponse{}, wrapError.ErrInvalidCredentials
	}

	passwordCleared, err := a.verifyLoginPassword(log, user, L.Password)
	if errors.Is(err, wrapError.ErrInvalidCredentials) {
		log.Warn("login failed",
			zap.String("username", L.Username),
			zap.String("user_id", user.ID),
			zap.String("reason", "invalid_credentials"),
		)
		return dto.LoginResponse{}, wrapError.ErrInvalidCredentials
	}
	if err != nil {
		return dto.LoginResponse{}, err
	}

	err = a.validateCredentials(L)
	if err != nil {
		log.Warn("login failed",
			zap.String("username", L.Username),
			zap.String("reason", "invalid_credentials"),
		)
		return dto.LoginResponse{}, wrapError.ErrInvalidCredentials
	}

	token, err := a.JwtService.AccessToken(user.ID, user.OrganisationID)
	if err != nil {
		log.Error("login failed",
			zap.String("user_id", user.ID),
			zap.String("organisation_id", user.OrganisationID),
			zap.String("reason", "access_token"),
			zap.Error(err),
		)
		return dto.LoginResponse{}, wrapError.ErrLoginFailed
	}

	refreshID, err := a.JwtService.FindIDByUserID(user.ID)
	if err != nil {
		log.Error("login failed",
			zap.String("user_id", user.ID),
			zap.String("organisation_id", user.OrganisationID),
			zap.String("reason", "refresh_token"),
			zap.Error(err),
		)
		return dto.LoginResponse{}, wrapError.ErrLoginFailed
	}

	var refreshToken string
	refreshAction := "create"
	if refreshID != "" {
		refreshAction = "update"
		refreshToken, err = a.JwtService.UpdateRefreshToken(refreshID, user.ID, user.OrganisationID)
		if err != nil {
			log.Error("login failed",
				zap.String("user_id", user.ID),
				zap.String("organisation_id", user.OrganisationID),
				zap.String("reason", "refresh_token"),
				zap.Error(err),
			)
			return dto.LoginResponse{}, wrapError.ErrLoginFailed
		}
	} else {
		claims, err := a.JwtService.RefreshToken(user.OrganisationID, user.ID, "")
		if err != nil {
			log.Error("login failed",
				zap.String("user_id", user.ID),
				zap.String("organisation_id", user.OrganisationID),
				zap.String("reason", "refresh_token"),
				zap.Error(err),
			)
			return dto.LoginResponse{}, wrapError.ErrLoginFailed
		}
		err = a.JwtService.InsertRefreshToken(claims.RefereshToken, claims.ExpiresAt, user.ID, claims.JTI)
		if err != nil {
			log.Error("login failed",
				zap.String("user_id", user.ID),
				zap.String("organisation_id", user.OrganisationID),
				zap.String("reason", "refresh_token"),
				zap.Error(err),
			)
			return dto.LoginResponse{}, wrapError.ErrLoginFailed
		}
		refreshToken = claims.RefereshToken
	}

	err = a.Repo.UpdateLastLoginAttempt(log, user.ID, user.LastLoginAttempt+1)
	if err != nil {
		log.Error("login failed",
			zap.String("user_id", user.ID),
			zap.String("reason", "last_login_update"),
			zap.Error(err),
		)
		return dto.LoginResponse{}, wrapError.ErrLoginFailed
	}

	log.Info("login success",
		zap.String("user_id", user.ID),
		zap.String("organisation_id", user.OrganisationID),
		zap.String("role_id", user.RoleID),
		zap.String("refresh_action", refreshAction),
	)

	access, err := a.loadRoleAccess(log, user.RoleID)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	response := dto.LoginResponse{}
	response.UserID = user.ID
	response.Token = token
	response.RefreshToken = refreshToken
	response.OrganisationID = user.OrganisationID
	response.RoleID = user.RoleID
	response.IsAdmin = access.IsAdmin
	response.Permissions = access.Permissions
	response.PasswordCleared = passwordCleared
	response.Message = "Login successful"
	return response, nil
}

func (a *UserService) loadRoleAccess(log *zap.Logger, roleID string) (rpdto.RoleAccess, error) {
	if a.RolePermissionSvc == nil {
		return rpdto.RoleAccess{Permissions: []rpdto.RoleModulePermission{}}, nil
	}
	access, err := a.RolePermissionSvc.FindModulesByRoleID(roleID)
	if err != nil {
		log.Error("login failed",
			zap.String("role_id", roleID),
			zap.String("reason", "role_permissions"),
			zap.Error(err),
		)
		return rpdto.RoleAccess{}, wrapError.ErrLoginFailed
	}
	return access, nil
}

func (a *UserService) verifyLoginPassword(log *zap.Logger, user *employee.User, password string) (bool, error) {
	if user.TempPassword != "" && user.TempPassword == password {
		if err := a.Repo.ClearTempPassword(log, user.ID); err != nil {
			log.Error("login failed",
				zap.String("user_id", user.ID),
				zap.String("reason", "clear_temp_password"),
				zap.Error(err),
			)
			return false, wrapError.ErrLoginFailed
		}
		return true, nil
	}
	verified, err := a.comparePwd(user.PasswordHash, password)
	if err != nil || !verified {
		return false, wrapError.ErrInvalidCredentials
	}
	return false, nil
}

func (a *UserService) validateCredentials(L dto.LoginUser) error {
	if L.Password == "" || L.Username == "" {
		return errors.New("invalid credentials")
	}
	return nil
}

func (a *UserService) comparePwd(DbPwd string, userPwd string) (verified bool, err error) {
	err = bcrypt.CompareHashAndPassword([]byte(DbPwd), []byte(userPwd))
	if err != nil {
		err = errors.New("password verification failed")
		return
	}
	return true, nil
}

func (a *UserService) RefreshToken(log *zap.Logger, refreshToken string) (dto.LoginResponse, error) {
	log = ensureLog(log)
	tokenresp, err := a.JwtService.ValidateRefreshToken(log, refreshToken)
	if err != nil {
		if errors.Is(err, wrapError.ErrRefreshSession) {
			return dto.LoginResponse{}, wrapError.ErrSessionExpired
		}
		return dto.LoginResponse{}, wrapError.ErrRefreshFailed
	}
	return a.toLoginResp(tokenresp), nil
}

func (a *UserService) Logout(log *zap.Logger, refreshToken string) error {
	log = ensureLog(log)
	return a.JwtService.LogoutRefreshToken(log, refreshToken)
}

func (a *UserService) toLoginResp(tokenresp jwt.TokenResp) dto.LoginResponse {
	return dto.LoginResponse{
		Token:        tokenresp.AccessToken,
		RefreshToken: tokenresp.RefreshToken,
	}
}

func (a *UserService) UpdatePassword(log *zap.Logger, req dto.UpdatePasswordRequest) error {
	log = ensureLog(log)
	req.Token = normalizeResetToken(req.Token)
	req.Password = strings.TrimSpace(req.Password)
	req.ConfirmPassword = strings.TrimSpace(req.ConfirmPassword)

	if req.Token == "" {
		log.Warn("password update failed", zap.String("reason", "missing_token"))
		return wrapError.ErrInvalidRequest
	}
	if err := a.validateNewPassword(req.Password, req.ConfirmPassword); err != nil {
		log.Warn("password update failed", zap.String("reason", "invalid_request"))
		return wrapError.ErrInvalidRequest
	}

	// Plain token from the reset link is hashed before lookup — DB stores SHA-256(plain_token), never the raw token.
	tokenHash := hashPasswordResetToken(req.Token)
	user, err := a.Repo.GetUserByPasswordResetTokenHash(log, tokenHash)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			log.Warn("password update failed", zap.String("reason", "invalid_token"))
			return wrapError.ErrInvalidRequest
		}
		log.Error("password update failed",
			zap.String("reason", "user_lookup"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordUpdateFailed
	}

	return a.persistPassword(log, user.ID, req.Password)
}

func (a *UserService) UpdatePasswordFirstLogin(log *zap.Logger, userID string, req dto.FirstLoginPasswordRequest) error {
	log = ensureLog(log)
	req.Password = strings.TrimSpace(req.Password)
	req.ConfirmPassword = strings.TrimSpace(req.ConfirmPassword)

	if userID == "" {
		log.Warn("first-login password update failed", zap.String("reason", "missing_user"))
		return wrapError.ErrInvalidRequest
	}
	if err := a.validateNewPassword(req.Password, req.ConfirmPassword); err != nil {
		log.Warn("first-login password update failed",
			zap.String("user_id", userID),
			zap.String("reason", "invalid_request"),
		)
		return wrapError.ErrInvalidRequest
	}

	user, err := a.Repo.GetUserByID(log, userID)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			log.Warn("first-login password update failed",
				zap.String("user_id", userID),
				zap.String("reason", "user_not_found"),
			)
			return wrapError.ErrInvalidRequest
		}
		log.Error("first-login password update failed",
			zap.String("user_id", userID),
			zap.String("reason", "user_lookup"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordUpdateFailed
	}
	if strings.TrimSpace(user.PasswordHash) != "" {
		log.Warn("first-login password update failed",
			zap.String("user_id", userID),
			zap.String("reason", "password_already_set"),
		)
		return wrapError.ErrInvalidRequest
	}

	return a.persistPassword(log, user.ID, req.Password)
}

func (a *UserService) persistPassword(log *zap.Logger, userID, password string) error {
	passwordHash, err := a.hashPassword(password)
	if err != nil {
		log.Error("password update failed",
			zap.String("user_id", userID),
			zap.String("reason", "hash"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordUpdateFailed
	}
	if err := a.Repo.UpdatePassword(log, userID, passwordHash); err != nil {
		log.Error("password update failed",
			zap.String("user_id", userID),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordUpdateFailed
	}
	log.Info("password update success", zap.String("user_id", userID))
	return nil
}

func (a *UserService) RequestPasswordReset(log *zap.Logger, emailID string) error {
	log = ensureLog(log)
	emailID = strings.ToLower(strings.TrimSpace(emailID))
	if emailID == "" {
		log.Warn("password reset request invalid", zap.String("reason", "missing_email"))
		return wrapError.ErrInvalidRequest
	}

	user, err := a.Repo.GetUserID(log, emailID)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			// Do not reveal whether the email exists.
			log.Info("password reset request completed",
				zap.String("reason", "user_not_found"),
				zap.String("email_id", emailID),
			)
			return nil
		}
		log.Error("password reset failed",
			zap.String("email_id", emailID),
			zap.String("reason", "user_lookup"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordResetFailed
	}

	cooldown := time.Duration(constants.PasswordResetCooldownMinutes) * time.Minute
	if user.LastPwdUpdated != nil && time.Since(*user.LastPwdUpdated) < cooldown {
		log.Warn("password reset blocked",
			zap.String("user_id", user.ID),
			zap.String("reason", "cooldown"),
			zap.Time("last_pwd_updated", *user.LastPwdUpdated),
		)
		return wrapError.ErrPasswordResetTooSoon
	}

	plainToken, tokenHash, err := createPasswordResetToken()
	if err != nil {
		log.Error("password reset failed",
			zap.String("user_id", user.ID),
			zap.String("reason", "token_generate"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordResetFailed
	}

	now := time.Now().UTC()
	if err := a.Repo.SavePasswordResetToken(log, user.ID, tokenHash, now); err != nil {
		log.Error("password reset failed",
			zap.String("user_id", user.ID),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordResetFailed
	}

	resetURL := a.buildPasswordResetURL(plainToken)
	employeeName := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if employeeName == "" {
		employeeName = user.EmailID
	}

	if a.Notifications == nil {
		log.Error("password reset failed",
			zap.String("user_id", user.ID),
			zap.String("reason", "notifications_unavailable"),
		)
		return wrapError.ErrPasswordResetFailed
	}

	err = a.Notifications.Create(context.Background(), notificationdto.CreateRequest{
		NotificationType: constants.PasswordResetEvent,
		Subject:          constants.PasswordResetSubject,
		Data: map[string]interface{}{
			"employee_name":    employeeName,
			"employee_email":   user.EmailID,
			"employee_id":      user.ID,
			"organisation_id":  user.OrganisationID,
			"hospital_name":    "Hospital Portal",
			"reset_url":        resetURL,
			"cooldown_minutes": fmt.Sprintf("%d", constants.PasswordResetCooldownMinutes),
		},
	})
	if err != nil {
		log.Error("password reset failed",
			zap.String("user_id", user.ID),
			zap.String("reason", "enqueue_email"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordResetFailed
	}

	log.Info("password reset link sent",
		zap.String("user_id", user.ID),
		zap.String("organisation_id", user.OrganisationID),
	)
	return nil
}

func (a *UserService) buildPasswordResetURL(token string) string {
	base := constants.DefaultPasswordResetBaseURL
	if a.cfg != nil && strings.TrimSpace(a.cfg.PasswordResetBaseURL) != "" {
		base = strings.TrimRight(a.cfg.PasswordResetBaseURL, "/")
	}
	return fmt.Sprintf("%s%s?token=%s", base, constants.PasswordResetPath, url.QueryEscape(token))
}

func createPasswordResetToken() (plainToken string, tokenHash string, err error) {
	raw := make([]byte, resetTokenBytes)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	plainToken = hex.EncodeToString(raw)
	tokenHash = hashPasswordResetToken(plainToken)
	return plainToken, tokenHash, nil
}

func hashPasswordResetToken(plainToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(plainToken)))
	return hex.EncodeToString(sum[:])
}

// normalizeResetToken accepts the plain token from the reset link (or a full reset URL)
// and returns the token value to hash for DB lookup.
func normalizeResetToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for i := 0; i < 2; i++ {
		decoded, err := url.QueryUnescape(raw)
		if err != nil || decoded == raw {
			break
		}
		raw = strings.TrimSpace(decoded)
	}
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			if token := strings.TrimSpace(u.Query().Get("token")); token != "" {
				return normalizeResetToken(token)
			}
		}
	}
	if idx := strings.Index(raw, "token="); idx >= 0 {
		fragment := raw[idx+len("token="):]
		if amp := strings.Index(fragment, "&"); amp >= 0 {
			fragment = fragment[:amp]
		}
		return normalizeResetToken(fragment)
	}
	return raw
}

func (a *UserService) validateNewPassword(password, confirmPassword string) error {
	if password == "" || confirmPassword == "" {
		return wrapError.ErrInvalidRequest
	}
	if password != confirmPassword {
		return wrapError.ErrInvalidRequest
	}
	if len(password) < minPasswordLen {
		return wrapError.ErrInvalidRequest
	}
	return nil
}

func (a *UserService) hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}
