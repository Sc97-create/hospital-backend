package payments

import (
	"errors"
	"fmt"
	"strings"

	centraldto "hospital-backend/central/payments/dto"
	"hospital-backend/internal/payments/providers/razorpay"
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
	Razorpay       *razorpay.RazorpayConfig
}
type PaymentController interface {
	RazorPayWebhook(c *fiber.Ctx) error
	UpdatePaymentManually(c *fiber.Ctx) error
	CreateOrder(c *fiber.Ctx) error
}

func NewPaymentController(payment PaymentServicer, webhook WebhookServicer, rzp *razorpay.RazorpayConfig) *IPayment {
	return &IPayment{PaymentService: payment, WebhookService: webhook, Razorpay: rzp}
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

// CreateOrder is the internal API that creates a Razorpay order (server-side Orders API).
// Protected by Basic Auth (see RegisterPaymentRoutes).
func (controller *IPayment) CreateOrder(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	if controller.Razorpay == nil || controller.Razorpay.Client == nil {
		logger.Error("order create failed", zap.String("reason", "razorpay_not_configured"))
		return wrapErrors.Wrap(wrapErrors.ErrOrderCreateFailed, c, fiber.StatusInternalServerError)
	}

	var req centraldto.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("order create request invalid", zap.Error(err))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.Receipt = strings.TrimSpace(req.Receipt)

	if req.Amount <= 0 {
		logger.Warn("order create failed", zap.String("reason", "invalid_amount"), zap.Int64("amount", req.Amount))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidAmount, c, fiber.StatusBadRequest)
	}
	if req.Currency == "" || req.Currency != "INR" {
		logger.Warn("order create failed", zap.String("reason", "invalid_currency"), zap.String("currency", req.Currency))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidCurrency, c, fiber.StatusBadRequest)
	}

	// Amount is already in subunits (paise).
	amountPaise := req.Amount
	logger.Info("order create attempt",
		zap.Int64("amount_paise", amountPaise),
		zap.String("currency", req.Currency),
		zap.String("receipt", req.Receipt),
	)

	body, err := controller.Razorpay.CreateOrder(c.Context(), amountPaise, req.Currency, req.Receipt, req.Notes)
	if err != nil {
		logger.Error("order create failed", zap.String("reason", "razorpay_create"), zap.Error(err))
		return wrapErrors.Wrap(wrapErrors.ErrOrderCreateFailed, c, fiber.StatusInternalServerError)
	}

	orderID := asString(body["id"])
	if orderID == "" {
		logger.Error("order create failed", zap.String("reason", "empty_order_id"))
		return wrapErrors.Wrap(wrapErrors.ErrOrderCreateFailed, c, fiber.StatusInternalServerError)
	}

	amount := asInt64(body["amount"])
	resp := centraldto.CreateOrderResponse{
		ID:         orderID,
		Entity:     asString(body["entity"]),
		Amount:     amount,
		AmountPaid: asInt64(body["amount_paid"]),
		AmountDue:  asInt64(body["amount_due"]),
		Currency:   asString(body["currency"]),
		Receipt:    asString(body["receipt"]),
		Status:     asString(body["status"]),
		Attempts:   int(asInt64(body["attempts"])),
		Notes:      body["notes"],
		CreatedAt:  asInt64(body["created_at"]),
	}
	if resp.Entity == "" {
		resp.Entity = "order"
	}
	if resp.AmountDue == 0 && resp.AmountPaid == 0 {
		resp.AmountDue = amount
	}
	if offerID := asString(body["offer_id"]); offerID != "" {
		resp.OfferID = &offerID
	}
	logger.Info("order create success",
		zap.String("order_id", resp.ID),
		zap.Int64("amount", resp.Amount),
		zap.String("status", resp.Status),
	)
	return c.Status(fiber.StatusOK).JSON(resp)
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

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func asInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	default:
		return 0
	}
}
