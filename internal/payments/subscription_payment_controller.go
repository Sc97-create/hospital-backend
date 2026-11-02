package payments

import (
	"errors"

	"hospital-backend/internal/payments/dto"
	"hospital-backend/pkg/middleware"
	wrapErrors "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// SubscriptionPaymentHandler is the Basic Auth API the central subscription flow calls.
type SubscriptionPaymentHandler struct {
	Service *PaymentsService
}

func (h *SubscriptionPaymentHandler) Create(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	if h == nil || h.Service == nil {
		logger.Error("subscription payment failed", zap.String("reason", "service_missing"))
		return wrapErrors.Wrap(wrapErrors.ErrPaymentCreateFailed, c, fiber.StatusInternalServerError)
	}
	var req dto.SubscriptionPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("subscription payment request invalid", zap.Error(err))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	resp, err := h.Service.RecordSubscriptionPayment(logger, req)
	if err != nil {
		return wrapSubscriptionPaymentError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *SubscriptionPaymentHandler) Confirm(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	if h == nil || h.Service == nil {
		logger.Error("subscription payment confirm failed", zap.String("reason", "service_missing"))
		return wrapErrors.Wrap(wrapErrors.ErrPaymentConfirmFailed, c, fiber.StatusInternalServerError)
	}
	var req dto.ConfirmSubscriptionPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("subscription payment confirm request invalid", zap.Error(err))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	resp, err := h.Service.ConfirmSubscriptionPayment(logger, req)
	if err != nil {
		return wrapSubscriptionPaymentError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func wrapSubscriptionPaymentError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapErrors.ErrInvalidRequest),
		errors.Is(err, wrapErrors.ErrInvalidAmount),
		errors.Is(err, wrapErrors.ErrInvalidCurrency):
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapErrors.ErrPaymentNotFound):
		return wrapErrors.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapErrors.ErrPaymentConfirmFailed):
		return wrapErrors.Wrap(err, c, fiber.StatusInternalServerError)
	default:
		return wrapErrors.Wrap(wrapErrors.ErrPaymentCreateFailed, c, fiber.StatusInternalServerError)
	}
}
