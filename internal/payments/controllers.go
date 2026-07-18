package payments

import (
	"hospital-backend/pkg/constants"

	"github.com/gofiber/fiber/v2"
)

type IPayment struct {
	PaymentService *PaymentsService
	WebhookService *IWebhookService
}
type PaymentController interface {
	RazorPayWebhook(c *fiber.Ctx) error
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
