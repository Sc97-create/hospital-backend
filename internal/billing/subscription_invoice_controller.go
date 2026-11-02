package billing

import (
	"errors"

	"hospital-backend/internal/billing/dto"
	"hospital-backend/pkg/middleware"
	wrapErrors "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// SubscriptionInvoiceHandler is the Basic Auth API the central subscription flow calls.
type SubscriptionInvoiceHandler struct {
	Service *InvoiceServ
}

func (h *SubscriptionInvoiceHandler) Create(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	if h == nil || h.Service == nil {
		logger.Error("subscription invoice failed", zap.String("reason", "service_missing"))
		return wrapErrors.Wrap(wrapErrors.ErrInvoiceCreateFailed, c, fiber.StatusInternalServerError)
	}
	var req dto.SubscriptionInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("subscription invoice request invalid", zap.Error(err))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	resp, err := h.Service.CreateSubscriptionInvoice(logger, req)
	if err != nil {
		return wrapSubscriptionInvoiceError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func wrapSubscriptionInvoiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapErrors.ErrInvalidRequest), errors.Is(err, wrapErrors.ErrInvalidAmount):
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapErrors.ErrInvoiceAlreadyExists):
		return wrapErrors.Wrap(err, c, fiber.StatusConflict)
	default:
		return wrapErrors.Wrap(wrapErrors.ErrInvoiceCreateFailed, c, fiber.StatusInternalServerError)
	}
}
