package payments

import (
	"hospital-backend/pkg/constants"
	wrapErrors "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

type IPayment struct {
	PaymentService *PaymentsService
	WebhookService *IWebhookService
}
type PaymentController interface {
	RazorPayWebhook(c *fiber.Ctx) error
	UpdatePaymentManually(c *fiber.Ctx) error
}

func NewPaymentController(payment *PaymentsService, webhook *IWebhookService) *IPayment {
	return &IPayment{PaymentService: payment, WebhookService: webhook}
}

func (controller *IPayment) RazorPayWebhook(c *fiber.Ctx) error {
	// Razorpay sends: X-Razorpay-Signature (HMAC-SHA256 hex of raw body)
	signature := c.Get("X-Razorpay-Signature")
	isverified, err := controller.WebhookService.ProcessWebhook(c.Body(), signature, constants.ProviderNameRazorpay)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to process webhook",
			"error":   err.Error(),
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
	payload, err := params.New(c)
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	invoiceID, err := payload.Getstring("invoice_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	paymentMode, err := payload.Getstring("payment_mode")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	txnRef, _ := payload.Getstring("transaction_reference")

	err = controller.PaymentService.ConfirmManualPayment(invoiceID, paymentMode, txnRef)
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "payment confirmed",
	})
}
