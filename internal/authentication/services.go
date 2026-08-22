package authentication

import (
	"errors"
	"hospital-backend/internal/authentication/dto"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/jwt"
	jwtAuth "hospital-backend/internal/jwt"
	"hospital-backend/internal/rolepermissions"
	rpdto "hospital-backend/internal/rolepermissions/dto"
	wrapError "hospital-backend/shared/error"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost     = 8
	minPasswordLen = 8
)

type UserService struct {
	Repo              UserRepository
	JwtService        jwtAuth.JwtService
	RolePermissionSvc *rolepermissions.RolePermissionService
}

func NewService(repo AuthRepo, jwtService jwt.JwtService, rolePermSvc *rolepermissions.RolePermissionService) UserService {
	return UserService{Repo: &repo, JwtService: jwtService, RolePermissionSvc: rolePermSvc}
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

	refreshID, err := a.JwtService.RefreshtokenRepo.FindIDByUserID(user.ID)
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

func (a *UserService) UpdatePassword(log *zap.Logger, userID string, req dto.UpdatePasswordRequest) error {
	log = ensureLog(log)
	req.Password = strings.TrimSpace(req.Password)
	req.ConfirmPassword = strings.TrimSpace(req.ConfirmPassword)
	if err := a.validateNewPassword(req); err != nil {
		log.Warn("password update failed",
			zap.String("user_id", userID),
			zap.String("reason", "invalid_request"),
		)
		return wrapError.ErrInvalidRequest
	}
	passwordHash, err := a.hashPassword(req.Password)
	if err != nil {
		log.Error("password update failed",
			zap.String("user_id", userID),
			zap.String("reason", "hash"),
			zap.Error(err),
		)
		return wrapError.ErrPasswordUpdateFailed
	}
	err = a.Repo.UpdatePassword(log, userID, passwordHash)
	if err != nil {
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

func (a *UserService) validateNewPassword(req dto.UpdatePasswordRequest) error {
	if req.Password == "" || req.ConfirmPassword == "" {
		return wrapError.ErrInvalidRequest
	}
	if req.Password != req.ConfirmPassword {
		return wrapError.ErrInvalidRequest
	}
	if len(req.Password) < minPasswordLen {
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
