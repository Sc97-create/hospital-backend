package customers

import (
	"errors"
	"strings"

	dto "hospital-backend/central/customers/dto"
	"hospital-backend/central/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type CustomerServicer interface {
	Signup(log *zap.Logger, payload dto.SignupPayload) (dto.SignupResult, error)
	VerifyEmail(log *zap.Logger, customerID string, payload dto.VerifyEmailPayload) error
}

type CustomerController struct {
	Service CustomerServicer
}

type ICustomerController interface {
	Signup(c *fiber.Ctx) error
	VerifyEmail(c *fiber.Ctx) error
}

func NewICustomerController(service CustomerServicer) ICustomerController {
	return &CustomerController{Service: service}
}

func (cc *CustomerController) Signup(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	payload, err := bindSignupPayload(c)
	if err != nil {
		logger.Warn("customer signup request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("customer signup attempt", zap.String("work_email", payload.WorkEmail))

	result, err := cc.Service.Signup(logger, payload)
	if err != nil {
		return wrapSignupError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "customer signed up successfully",
		"customer_id":  result.CustomerID,
		"work_email":   result.WorkEmail,
		"status":       result.Status,
		"access_token": result.AccessToken,
	})
}

func (cc *CustomerController) VerifyEmail(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	customerID := middleware.GetUserID(c)
	if customerID == "" {
		logger.Warn("customer email verify failed", zap.String("reason", "missing_user"))
		return wrapError.Wrap(wrapError.ErrSessionExpired, c, fiber.StatusUnauthorized)
	}

	payload, err := bindVerifyEmailPayload(c)
	if err != nil {
		logger.Warn("customer email verify request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("customer email verify attempt", zap.String("customer_id", customerID))
	if err := cc.Service.VerifyEmail(logger, customerID, payload); err != nil {
		return wrapVerifyError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "email verified successfully",
	})
}

func bindSignupPayload(c *fiber.Ctx) (dto.SignupPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.SignupPayload{}, err
	}
	payload := dto.SignupPayload{}
	if payload.FullName, err = requiredString(p, "full_name"); err != nil {
		return payload, err
	}
	if payload.WorkEmail, err = requiredString(p, "work_email"); err != nil {
		return payload, err
	}
	if payload.Password, err = requiredString(p, "password"); err != nil {
		return payload, err
	}
	return payload, nil
}

func bindVerifyEmailPayload(c *fiber.Ctx) (dto.VerifyEmailPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.VerifyEmailPayload{}, err
	}
	payload := dto.VerifyEmailPayload{}
	if payload.Code, err = requiredString(p, "code"); err != nil {
		return payload, err
	}
	return payload, nil
}

func requiredString(p *params.Payload, key string) (string, error) {
	v, err := p.Getstring(key)
	if err != nil || strings.TrimSpace(v) == "" {
		return "", wrapError.ErrInvalidRequest
	}
	return strings.TrimSpace(v), nil
}

func wrapSignupError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrCustomerAlreadyExists):
		return wrapError.Wrap(err, c, fiber.StatusConflict)
	default:
		return wrapError.Wrap(wrapError.ErrCustomerCreateFailed, c, fiber.StatusInternalServerError)
	}
}

func wrapVerifyError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrInvalidVerificationCode):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrVerificationCodeExpired):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapError.Wrap(wrapError.ErrCustomerVerifyFailed, c, fiber.StatusInternalServerError)
	}
}
