package subscriptions

import (
	"errors"
	"strings"

	dto "hospital-backend/central/subscriptions/dto"
	"hospital-backend/central/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type SubscriptionController struct {
	Service *SubscriptionService
}

type ISubscriptionController interface {
	Create(c *fiber.Ctx) error
	ConfirmCheckout(c *fiber.Ctx) error
	CheckEnd(c *fiber.Ctx) error
}

func NewISubscriptionController(service *SubscriptionService) ISubscriptionController {
	return &SubscriptionController{Service: service}
}

func (sc *SubscriptionController) Create(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	payload, err := bindCreatePayload(c)
	if err != nil {
		logger.Warn("subscription create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("subscription create attempt",
		zap.String("tenant_id", payload.TenantID),
		zap.String("plan_id", payload.PlanID),
		zap.Int("billing_cycle", payload.BillingCycle),
	)

	result, err := sc.Service.CreateSubscription(logger, c.Context(), payload)
	if err != nil {
		return wrapCreateError(c, err)
	}
	resp := fiber.Map{
		"message":         "subscription created successfully",
		"subscription_id": result.SubscriptionID,
		"tenant_id":       result.TenantID,
		"hospital_name":   result.HospitalName,
		"price":           result.Price,
		"billing_cycle":   result.BillingCycle,
		"status":          result.Status,
		"start_at":        result.StartAt,
		"end_at":          result.EndAt,
	}
	if result.OrderID != "" {
		resp["order_id"] = result.OrderID
	}
	if result.KeyID != "" {
		resp["key_id"] = result.KeyID
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (sc *SubscriptionController) ConfirmCheckout(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	payload, err := bindConfirmCheckoutPayload(c)
	if err != nil {
		logger.Warn("subscription checkout confirm request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("subscription checkout confirm attempt",
		zap.String("order_id", payload.RazorpayOrderID),
		zap.String("payment_id", payload.RazorpayPaymentID),
	)

	result, err := sc.Service.ConfirmCheckout(logger, payload)
	if err != nil {
		return wrapConfirmCheckoutError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":         "payment details saved",
		"subscription_id": result.SubscriptionID,
		"order_id":        result.OrderID,
		"payment_id":      result.PaymentID,
		"status":          result.Status,
	})
}

func (sc *SubscriptionController) CheckEnd(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := strings.TrimSpace(c.Params("organisation_id"))
	if organisationID == "" {
		logger.Warn("subscription end check request invalid", zap.String("reason", "missing_organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("subscription end check attempt", zap.String("organisation_id", organisationID))
	result, err := sc.Service.CheckEndByOrganisation(logger, organisationID)
	if err != nil {
		return wrapCheckEndError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":         result.Message,
		"ended":           result.Ended,
		"organisation_id": result.OrganisationID,
		"end_at":          result.EndAt,
	})
}

func bindCreatePayload(c *fiber.Ctx) (dto.CreateSubscriptionPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.CreateSubscriptionPayload{}, err
	}
	payload := dto.CreateSubscriptionPayload{}
	if payload.PlanID, err = requiredString(p, "plan_id"); err != nil {
		return payload, err
	}
	if payload.TenantID, err = requiredString(p, "tenant_id"); err != nil {
		return payload, err
	}
	payload.HospitalName, _ = p.Getstring("hospital_name")
	payload.HospitalName = strings.TrimSpace(payload.HospitalName)
	// Optional for free trial; validated against allowed months for paid plans in the service.
	if cycle, err := p.Getint("billing_cycle"); err == nil {
		payload.BillingCycle = cycle
	}
	if price, err := p.Getfloat("price"); err == nil {
		payload.Price = price
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

func bindConfirmCheckoutPayload(c *fiber.Ctx) (dto.ConfirmCheckoutPayload, error) {
	p, err := params.New(c)
	if err != nil {
		return dto.ConfirmCheckoutPayload{}, err
	}
	payload := dto.ConfirmCheckoutPayload{}
	if payload.RazorpayPaymentID, err = requiredString(p, "razorpay_payment_id"); err != nil {
		return payload, err
	}
	if payload.RazorpayOrderID, err = requiredString(p, "razorpay_order_id"); err != nil {
		return payload, err
	}
	if payload.RazorpaySignature, err = requiredString(p, "razorpay_signature"); err != nil {
		return payload, err
	}
	return payload, nil
}

func wrapCreateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest),
		errors.Is(err, wrapError.ErrInvalidAmount),
		errors.Is(err, wrapError.ErrInvalidCurrency):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrPlanNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrUnauthorized):
		return wrapError.Wrap(err, c, fiber.StatusUnauthorized)
	case errors.Is(err, wrapError.ErrOrderCreateFailed):
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	default:
		return wrapError.Wrap(wrapError.ErrSubscriptionCreateFailed, c, fiber.StatusInternalServerError)
	}
}

func wrapCheckEndError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrOrganisationNotFound),
		errors.Is(err, wrapError.ErrSubscriptionNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	default:
		return wrapError.Wrap(wrapError.ErrOrganisationFetchFailed, c, fiber.StatusInternalServerError)
	}
}

func wrapConfirmCheckoutError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	case errors.Is(err, wrapError.ErrSubscriptionNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	default:
		return wrapError.Wrap(wrapError.ErrSubscriptionUpdateFailed, c, fiber.StatusInternalServerError)
	}
}
