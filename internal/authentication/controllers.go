package authentication

import (
	"errors"
	"hospital-backend/internal/authentication/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AuthServicer interface {
	Login(log *zap.Logger, req dto.LoginUser) (dto.LoginResponse, error)
	RefreshToken(log *zap.Logger, refreshToken string) (dto.LoginResponse, error)
	Logout(log *zap.Logger, refreshToken string) error
	UpdatePassword(log *zap.Logger, userID string, req dto.UpdatePasswordRequest) error
}

type AuthController struct {
	AuthService AuthServicer
}

func NewAuthController(service AuthServicer) *AuthController {
	return &AuthController{AuthService: service}
}

func (a *AuthController) Login(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("login request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	loginRequest := dto.LoginUser{}
	loginRequest.Username, err = payload.Getstring("user_name")
	if err != nil {
		logger.Warn("login request invalid", zap.Error(err), zap.String("field", "user_name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	loginRequest.Password, err = payload.Getstring("password")
	if err != nil {
		logger.Warn("login request invalid", zap.Error(err), zap.String("field", "password"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("login attempt", zap.String("username", loginRequest.Username))

	userResp, err := a.AuthService.Login(logger, loginRequest)
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, wrapError.ErrInvalidCredentials) {
			status = fiber.StatusUnauthorized
		}
		return wrapError.Wrap(err, c, status)
	}

	a.setRefreshToken(c, userResp.RefreshToken)
	return c.Status(fiber.StatusOK).JSON(userResp)
}

func (a *AuthController) Refresh(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		logger.Warn("refresh failed", zap.String("reason", "missing_cookie"))
		return wrapError.Wrap(wrapError.ErrSessionExpired, c, fiber.StatusUnauthorized)
	}

	logger.Info("refresh attempt", zap.Bool("has_refresh_cookie", true))

	tokenresp, err := a.AuthService.RefreshToken(logger, refreshToken)
	if err != nil {
		status := fiber.StatusInternalServerError
		clientErr := wrapError.ErrRefreshFailed
		if errors.Is(err, wrapError.ErrSessionExpired) {
			status = fiber.StatusUnauthorized
			clientErr = wrapError.ErrSessionExpired
		}
		return wrapError.Wrap(clientErr, c, status)
	}

	a.setRefreshToken(c, tokenresp.RefreshToken)
	response := make(map[string]any)
	response["accesstoken"] = tokenresp.Token
	response["message"] = "user logged in successfully"
	if err := c.Status(fiber.StatusOK).JSON(response); err != nil {
		logger.Error("refresh response failed", zap.Error(err))
		return err
	}
	return
}

func (a *AuthController) Logout(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	refreshToken := c.Cookies("refresh_token")
	logger.Info("logout attempt", zap.Bool("has_refresh_cookie", refreshToken != ""))

	_ = a.AuthService.Logout(logger, refreshToken)
	a.clearRefreshToken(c)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "logged out successfully",
	})
}

func (a *AuthController) UpdatePassword(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	userID := middleware.GetUserID(c)
	if userID == "" {
		logger.Warn("password update failed", zap.String("reason", "missing_user"))
		return wrapError.Wrap(wrapError.ErrSessionExpired, c, fiber.StatusUnauthorized)
	}

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("password update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	req := dto.UpdatePasswordRequest{}
	req.Password, err = payload.Getstring("password")
	if err != nil {
		logger.Warn("password update request invalid", zap.String("field", "password"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	req.ConfirmPassword, err = payload.Getstring("confirm_password")
	if err != nil {
		logger.Warn("password update request invalid", zap.String("field", "confirm_password"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("password update attempt", zap.String("user_id", userID))
	err = a.AuthService.UpdatePassword(logger, userID, req)
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, wrapError.ErrInvalidRequest) {
			status = fiber.StatusBadRequest
		}
		return wrapError.Wrap(err, c, status)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "password updated successfully"})
}

func (a *AuthController) setRefreshToken(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:  "refresh_token",
		Value: token,
		//HTTPOnly: true,
		//Secure:   false,
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/",
		MaxAge:   int(1440 * 60),
	})
}

func (a *AuthController) clearRefreshToken(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/",
		MaxAge:   -1,
	})
}
