package payments

import "github.com/gofiber/fiber/v2"

type IPayment struct {
	PaymentService *PaymentsService
}
type PaymentController interface {
	RazorPayWebhook(c *fiber.Ctx) error
}

func NewPaymentController(payment *PaymentsService) *IPayment {
	return &IPayment{PaymentService: payment}
}

func (controller *IPayment) RazorPayWebhook(c *fiber.Ctx) error {

	signature := c.Get("x-RazorPay-Signature")
	controller.PaymentService.ProcessWebhook(c.Body(), signature, ProviderNameRazorpay)
	return nil
}
