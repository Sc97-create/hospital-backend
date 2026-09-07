package payments

import (
	"errors"
	"hospital-backend/pkg/constants"
	"hospital-backend/pkg/middleware"
	wrapErrors "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type PaymentServicer interface {
	ConfirmManualPayment(log *zap.Logger, invoiceID, organisationID, paymentMode, txnRef string) error
}

type WebhookServicer interface {
	ProcessWebhook(log *zap.Logger, payload []byte, signature string, provider string) (bool, error)
}

type IPayment struct {
	PaymentService PaymentServicer
	WebhookService WebhookServicer
}
type PaymentController interface {
	RazorPayWebhook(c *fiber.Ctx) error
	UpdatePaymentManually(c *fiber.Ctx) error
}

func NewPaymentController(payment PaymentServicer, webhook WebhookServicer) *IPayment {
	return &IPayment{PaymentService: payment, WebhookService: webhook}
}

func (controller *IPayment) RazorPayWebhook(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	logger.Info("payment webhook received", zap.String("provider", constants.ProviderNameRazorpay))

	signature := c.Get("X-Razorpay-Signature")
	isverified, err := controller.WebhookService.ProcessWebhook(logger, c.Body(), signature, constants.ProviderNameRazorpay)
	if err != nil {
		if !isverified {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to process webhook",
		})
	}
	if !isverified {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Webhook processed successfully",
	})
}

func (controller *IPayment) UpdatePaymentManually(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("payment confirm request invalid", zap.Error(err))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	invoiceID, err := payload.Getstring("invoice_id")
	if err != nil || invoiceID == "" {
		logger.Warn("payment confirm request invalid", zap.String("field", "invoice_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	organisationID, err := payload.Getstring("organisation_id")
	if err != nil || organisationID == "" {
		logger.Warn("payment confirm request invalid", zap.String("field", "organisation_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	paymentMode, err := payload.Getstring("payment_mode")
	if err != nil || paymentMode == "" {
		logger.Warn("payment confirm request invalid", zap.String("field", "payment_mode"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	txnRef, _ := payload.Getstring("transaction_reference")

	logger.Info("payment confirm attempt",
		zap.String("invoice_id", invoiceID),
		zap.String("organisation_id", organisationID),
		zap.String("payment_mode", paymentMode),
	)

	err = controller.PaymentService.ConfirmManualPayment(logger, invoiceID, organisationID, paymentMode, txnRef)
	if err != nil {
		return controller.wrapConfirmError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "payment confirmed",
	})
}

func (controller *IPayment) wrapConfirmError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapErrors.ErrPaymentNotFound):
		return wrapErrors.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapErrors.ErrInvalidRequest),
		errors.Is(err, wrapErrors.ErrUnsupportedPaymentMode):
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapErrors.Wrap(wrapErrors.ErrPaymentConfirmFailed, c, fiber.StatusInternalServerError)
	}
}
