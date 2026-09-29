package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"hospital-backend/central/payments"
	paymentdto "hospital-backend/central/payments/dto"
	"hospital-backend/central/plans"
	dto "hospital-backend/central/subscriptions/dto"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PlanReader loads plan catalog rows for subscription pricing.
type PlanReader interface {
	GetByID(log *zap.Logger, planID string) (plans.Plan, error)
}

// OrderCreator creates a Razorpay order via central/payments.
type OrderCreator interface {
	CreateOrder(log *zap.Logger, ctx context.Context, req paymentdto.CreateOrderRequest) (paymentdto.CreateOrderResponse, error)
}

type SubscriptionService struct {
	DB     *gorm.DB
	Repo   SubscriptionRepository
	Plan   PlanReader
	Orders OrderCreator
	Tenant TenantActivator
	KeyID  string // Razorpay key_id for frontend checkout
}

// TenantActivator activates a tenant after payment/webhook (or free-trial) confirmation.
type TenantActivator interface {
	Activate(log *zap.Logger, tenantID string) error
}

func NewSubscriptionService(
	db *gorm.DB,
	repo SubscriptionRepository,
	planReader PlanReader,
	orders OrderCreator,
	tenantActivator TenantActivator,
	razorpayKeyID string,
) *SubscriptionService {
	return &SubscriptionService{
		DB:     db,
		Repo:   repo,
		Plan:   planReader,
		Orders: orders,
		Tenant: tenantActivator,
		KeyID:  strings.TrimSpace(razorpayKeyID),
	}
}

func (s *SubscriptionService) CreateSubscription(log *zap.Logger, ctx context.Context, payload dto.CreateSubscriptionPayload) (dto.CreateSubscriptionResult, error) {
	log = ensureLog(log)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateCreatePayload(payload); err != nil {
		log.Warn("subscription create failed", zap.String("reason", "validation"), zap.Error(err))
		return dto.CreateSubscriptionResult{}, wrapError.ErrInvalidRequest
	}

	planID := strings.TrimSpace(payload.PlanID)
	plan, err := s.Plan.GetByID(log, planID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("subscription create failed",
				zap.String("plan_id", planID),
				zap.String("reason", "plan_not_found"),
			)
			return dto.CreateSubscriptionResult{}, wrapError.ErrPlanNotFound
		}
		log.Error("subscription create failed",
			zap.String("plan_id", planID),
			zap.String("reason", "plan_lookup"),
			zap.Error(err),
		)
		return dto.CreateSubscriptionResult{}, wrapError.ErrSubscriptionCreateFailed
	}

	if !isFreeTrialPlan(plan) && !isAllowedBillingCycle(payload.BillingCycle) {
		log.Warn("subscription create failed",
			zap.String("plan_id", plan.ID),
			zap.Int("billing_cycle", payload.BillingCycle),
			zap.String("reason", "invalid_billing_cycle"),
		)
		return dto.CreateSubscriptionResult{}, wrapError.ErrInvalidRequest
	}

	sub := buildSubscription(plan, payload)
	if err := s.Repo.Create(log, nil, sub); err != nil {
		log.Error("subscription create failed",
			zap.String("tenant_id", payload.TenantID),
			zap.String("plan_id", plan.ID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return dto.CreateSubscriptionResult{}, wrapError.ErrSubscriptionCreateFailed
	}

	result := dto.CreateSubscriptionResult{
		SubscriptionID: sub.ID,
		TenantID:       sub.TenantID,
		HospitalName:   strings.TrimSpace(payload.HospitalName),
		Price:          sub.Price,
		BillingCycle:   sub.BillingCycle,
		Status:         sub.Status,
		StartAt:        sub.StartAt,
		EndAt:          sub.EndAt,
	}

	// Free trial has zero price — no Razorpay order; already active.
	if isFreeTrialPlan(plan) || sub.Price <= 0 {
		if err := s.activateTenant(log, sub.TenantID); err != nil {
			return dto.CreateSubscriptionResult{}, err
		}
		log.Info("subscription create success",
			zap.String("subscription_id", sub.ID),
			zap.String("tenant_id", sub.TenantID),
			zap.String("hospital_name", result.HospitalName),
			zap.String("status", sub.Status),
			zap.Float64("price", sub.Price),
		)
		return result, nil
	}

	if s.Orders == nil {
		log.Error("subscription create failed", zap.String("reason", "order_creator_missing"))
		return dto.CreateSubscriptionResult{}, wrapError.ErrOrderCreateFailed
	}

	amountPaise := payments.AmountInPaise(sub.Price)
	orderReq := paymentdto.CreateOrderRequest{
		Amount:   amountPaise,
		Currency: payments.CurrencyINR,
		Receipt:  truncateReceipt(sub.ID),
		Notes: map[string]string{
			"subscription_id": sub.ID,
			"tenant_id":       sub.TenantID,
			"hospital_name":   strings.TrimSpace(payload.HospitalName),
		},
	}

	orderResp, err := s.Orders.CreateOrder(log, ctx, orderReq)
	if err != nil {
		log.Error("subscription create order failed",
			zap.String("subscription_id", sub.ID),
			zap.Int64("amount_paise", amountPaise),
			zap.String("currency", payments.CurrencyINR),
			zap.Error(err),
		)
		return dto.CreateSubscriptionResult{}, err
	}

	if err := s.Repo.UpdateByID(log, nil, sub.ID, map[string]interface{}{
		"order_id": orderResp.ID,
	}); err != nil {
		log.Error("subscription create failed",
			zap.String("subscription_id", sub.ID),
			zap.String("order_id", orderResp.ID),
			zap.String("reason", "bind_order_id"),
			zap.Error(err),
		)
		return dto.CreateSubscriptionResult{}, wrapError.ErrSubscriptionUpdateFailed
	}

	result.OrderID = orderResp.ID
	result.KeyID = s.KeyID
	log.Info("subscription create success",
		zap.String("subscription_id", sub.ID),
		zap.String("tenant_id", sub.TenantID),
		zap.String("hospital_name", result.HospitalName),
		zap.String("status", sub.Status),
		zap.Float64("price", sub.Price),
		zap.String("order_id", result.OrderID),
	)
	return result, nil
}

// ActivateByOrderID marks a pending_payment subscription active after a verified
// Razorpay webhook (order.paid), then activates the tenant. Idempotent when already active.
func (s *SubscriptionService) ActivateByOrderID(log *zap.Logger, orderID, providerPaymentID string) error {
	log = ensureLog(log)
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return wrapError.ErrInvalidRequest
	}

	sub, err := s.Repo.GetByOrderID(log, orderID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			log.Warn("subscription activate failed",
				zap.String("order_id", orderID),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrSubscriptionNotFound
		}
		log.Error("subscription activate failed",
			zap.String("order_id", orderID),
			zap.String("reason", "lookup"),
			zap.Error(err),
		)
		return wrapError.ErrSubscriptionUpdateFailed
	}

	if sub.Status == statusActive {
		if pid := strings.TrimSpace(providerPaymentID); pid != "" && sub.ProviderPaymentID == "" {
			_ = s.Repo.UpdateByID(log, nil, sub.ID, map[string]interface{}{
				"provider_payment_id": pid,
			})
		}
		if err := s.activateTenant(log, sub.TenantID); err != nil {
			return err
		}
		log.Info("subscription activate skipped",
			zap.String("subscription_id", sub.ID),
			zap.String("order_id", orderID),
			zap.String("reason", "already_active"),
		)
		return nil
	}

	now := time.Now()
	start := now
	end := subscriptionEndAt(now, sub.BillingCycle)
	updates := map[string]interface{}{
		"status":   statusActive,
		"start_at": start,
		"end_at":   end,
	}
	if pid := strings.TrimSpace(providerPaymentID); pid != "" {
		updates["provider_payment_id"] = pid
	}

	if err := s.Repo.UpdateByID(log, nil, sub.ID, updates); err != nil {
		log.Error("subscription activate failed",
			zap.String("subscription_id", sub.ID),
			zap.String("order_id", orderID),
			zap.String("reason", "update"),
			zap.Error(err),
		)
		return wrapError.ErrSubscriptionUpdateFailed
	}

	if err := s.activateTenant(log, sub.TenantID); err != nil {
		return err
	}

	log.Info("subscription activate success",
		zap.String("subscription_id", sub.ID),
		zap.String("tenant_id", sub.TenantID),
		zap.String("order_id", orderID),
		zap.String("provider_payment_id", providerPaymentID),
	)
	return nil
}

// ConfirmCheckout stores Razorpay Checkout success fields from the frontend.
// Signature verification and tenant/subscription activation happen on webhook.
func (s *SubscriptionService) ConfirmCheckout(log *zap.Logger, payload dto.ConfirmCheckoutPayload) (dto.ConfirmCheckoutResult, error) {
	log = ensureLog(log)
	payload.RazorpayOrderID = strings.TrimSpace(payload.RazorpayOrderID)
	payload.RazorpayPaymentID = strings.TrimSpace(payload.RazorpayPaymentID)
	payload.RazorpaySignature = strings.TrimSpace(payload.RazorpaySignature)
	if payload.RazorpayOrderID == "" || payload.RazorpayPaymentID == "" || payload.RazorpaySignature == "" {
		log.Warn("subscription checkout store failed", zap.String("reason", "missing_fields"))
		return dto.ConfirmCheckoutResult{}, wrapError.ErrInvalidRequest
	}

	sub, err := s.Repo.GetByOrderID(log, payload.RazorpayOrderID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			log.Warn("subscription checkout store failed",
				zap.String("order_id", payload.RazorpayOrderID),
				zap.String("reason", "not_found"),
			)
			return dto.ConfirmCheckoutResult{}, wrapError.ErrSubscriptionNotFound
		}
		log.Error("subscription checkout store failed",
			zap.String("order_id", payload.RazorpayOrderID),
			zap.String("reason", "lookup"),
			zap.Error(err),
		)
		return dto.ConfirmCheckoutResult{}, wrapError.ErrSubscriptionUpdateFailed
	}

	if err := s.Repo.UpdateByID(log, nil, sub.ID, map[string]interface{}{
		"provider_payment_id": payload.RazorpayPaymentID,
		"payment_signature":   payload.RazorpaySignature,
	}); err != nil {
		log.Error("subscription checkout store failed",
			zap.String("subscription_id", sub.ID),
			zap.String("order_id", payload.RazorpayOrderID),
			zap.String("reason", "update"),
			zap.Error(err),
		)
		return dto.ConfirmCheckoutResult{}, wrapError.ErrSubscriptionUpdateFailed
	}

	log.Info("subscription checkout store success",
		zap.String("subscription_id", sub.ID),
		zap.String("order_id", payload.RazorpayOrderID),
		zap.String("payment_id", payload.RazorpayPaymentID),
		zap.String("status", sub.Status),
	)
	return dto.ConfirmCheckoutResult{
		SubscriptionID: sub.ID,
		OrderID:        payload.RazorpayOrderID,
		PaymentID:      payload.RazorpayPaymentID,
		Status:         sub.Status,
	}, nil
}

// CheckEndByOrganisation reports whether the latest subscription for the organisation's tenant
// has reached end_at. It is a standalone check and is not called from other flows.
func (s *SubscriptionService) CheckEndByOrganisation(log *zap.Logger, organisationID string) (dto.CheckEndResult, error) {
	log = ensureLog(log)
	organisationID = strings.TrimSpace(organisationID)
	if organisationID == "" {
		return dto.CheckEndResult{}, wrapError.ErrInvalidRequest
	}

	tenantID, err := s.tenantIDByOrganisation(log, organisationID)
	if err != nil {
		return dto.CheckEndResult{}, err
	}

	sub, err := s.Repo.GetLatestByTenantID(log, tenantID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			log.Warn("subscription end check failed",
				zap.String("organisation_id", organisationID),
				zap.String("tenant_id", tenantID),
				zap.String("reason", "subscription_not_found"),
			)
			return dto.CheckEndResult{}, wrapError.ErrSubscriptionNotFound
		}
		log.Error("subscription end check failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "lookup"),
			zap.Error(err),
		)
		return dto.CheckEndResult{}, wrapError.ErrSubscriptionUpdateFailed
	}

	result := periodEndResult(organisationID, sub, time.Now())
	log.Info("subscription end check success",
		zap.String("organisation_id", organisationID),
		zap.String("subscription_id", sub.ID),
		zap.Bool("ended", result.Ended),
		zap.String("message", result.Message),
	)
	return result, nil
}

func (s *SubscriptionService) tenantIDByOrganisation(log *zap.Logger, organisationID string) (string, error) {
	if s.DB == nil {
		log.Error("subscription end check failed", zap.String("reason", "db_missing"))
		return "", wrapError.ErrOrganisationFetchFailed
	}
	var row struct {
		TenantID string `gorm:"column:tenant_id"`
	}
	err := s.DB.Table("organisations").Select("tenant_id").Where("id = ?", organisationID).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("subscription end check failed",
				zap.String("organisation_id", organisationID),
				zap.String("reason", "organisation_not_found"),
			)
			return "", wrapError.ErrOrganisationNotFound
		}
		log.Error("subscription end check failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "organisation_lookup"),
			zap.Error(err),
		)
		return "", wrapError.ErrOrganisationFetchFailed
	}
	if strings.TrimSpace(row.TenantID) == "" {
		return "", wrapError.ErrOrganisationNotFound
	}
	return row.TenantID, nil
}

func periodEndResult(organisationID string, sub Subscription, now time.Time) dto.CheckEndResult {
	ended := sub.EndAt != nil && !sub.EndAt.After(now)
	message := "active"
	if ended && sub.BillingCycle == billingCycleTrial {
		message = "trial period ended"
	}
	if ended && sub.BillingCycle != billingCycleTrial {
		message = "subscription ended"
	}
	return dto.CheckEndResult{
		OrganisationID: organisationID,
		Message:        message,
		Ended:          ended,
		EndAt:          sub.EndAt,
	}
}

func (s *SubscriptionService) activateTenant(log *zap.Logger, tenantID string) error {
	if s.Tenant == nil {
		log.Error("tenant activate failed", zap.String("tenant_id", tenantID), zap.String("reason", "activator_missing"))
		return wrapError.ErrTenantUpdateFailed
	}
	if err := s.Tenant.Activate(log, tenantID); err != nil {
		return err
	}
	return nil
}

func subscriptionEndAt(start time.Time, billingCycle string) time.Time {
	if billingCycle == billingCycleTrial {
		return start.AddDate(0, 0, freeTrialDurationDays)
	}
	months, err := strconv.Atoi(strings.TrimSpace(billingCycle))
	if err != nil || months <= 0 {
		return start.AddDate(0, 1, 0)
	}
	return start.AddDate(0, months, 0)
}

func truncateReceipt(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 40 {
		return id
	}
	return id[:40]
}

func buildSubscription(plan plans.Plan, payload dto.CreateSubscriptionPayload) Subscription {
	now := time.Now()
	sub := Subscription{
		ID:        uuid.NewString(),
		TenantID:  strings.TrimSpace(payload.TenantID),
		PlanID:    plan.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if isFreeTrialPlan(plan) {
		start := now
		end := now.AddDate(0, 0, freeTrialDurationDays)
		sub.Status = statusActive
		sub.BillingCycle = billingCycleTrial
		sub.Price = 0
		sub.StartAt = &start
		sub.EndAt = &end
		return sub
	}

	months := payload.BillingCycle
	sub.Status = statusPendingPayment
	sub.BillingCycle = fmt.Sprintf("%d", months)
	sub.Price = totalSubscriptionPrice(plan, payload.Price, months)
	// Start/end are set when order.paid activates the subscription.
	return sub
}

func isFreeTrialPlan(plan plans.Plan) bool {
	return strings.EqualFold(strings.TrimSpace(plan.Name), plans.PlanNameFreeTrial)
}

// totalSubscriptionPrice multiplies the monthly unit by billing months.
// Unit price prefers payload price; otherwise plan monthly price after discount %.
func totalSubscriptionPrice(plan plans.Plan, payloadPrice float64, months int) float64 {
	unit := payloadPrice
	if unit <= 0 {
		unit = plan.MonthlyPrice * (1 - plan.Discount/100)
	}
	return unit * float64(months)
}
