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

// SubscriptionBilling calls the hospital invoice and payment APIs over HTTP.
// Those packages are not imported here.
type SubscriptionBilling interface {
	CreateSubscriptionInvoice(log *zap.Logger, ctx context.Context, req paymentdto.SubscriptionInvoiceRequest) (paymentdto.SubscriptionInvoiceResponse, error)
	RecordSubscriptionPayment(log *zap.Logger, ctx context.Context, req paymentdto.SubscriptionPaymentRequest) error
	ConfirmSubscriptionPayment(log *zap.Logger, ctx context.Context, req paymentdto.ConfirmSubscriptionPaymentRequest) (paymentdto.ConfirmSubscriptionPaymentResponse, error)
}

type SubscriptionService struct {
	DB      *gorm.DB
	Repo    SubscriptionRepository
	Plan    PlanReader
	Orders  OrderCreator
	Billing SubscriptionBilling
	Tenant  TenantActivator
	KeyID   string // Razorpay key_id for frontend checkout
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
	if s.Plan == nil || s.Repo == nil {
		log.Error("subscription create failed", zap.String("reason", "missing_deps"))
		return dto.CreateSubscriptionResult{}, wrapError.ErrSubscriptionCreateFailed
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

	orderID, err := s.openPaidSubscription(log, ctx, paidSubscription{
		Sub:            sub,
		HospitalName:   strings.TrimSpace(payload.HospitalName),
		CustomerID:     payload.CustomerID,
		OrganisationID: payload.OrganisationID,
		Price:          sub.Price,
	})
	if err != nil {
		return dto.CreateSubscriptionResult{}, err
	}
	result.OrderID = orderID
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

// UpdateSubscription changes the plan, billing cycle, and period on an existing subscription.
// A standard plan creates a Razorpay order, a subscription invoice, and a card payment.
// A free trial is activated immediately and does not create an order.
func (s *SubscriptionService) UpdateSubscription(log *zap.Logger, ctx context.Context, payload dto.UpdateSubscriptionPayload) (dto.CreateSubscriptionResult, error) {
	log = ensureLog(log)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateUpdatePayload(payload); err != nil {
		log.Warn("subscription update failed", zap.String("reason", "validation"))
		return dto.CreateSubscriptionResult{}, wrapError.ErrInvalidRequest
	}
	if s.Plan == nil || s.Repo == nil {
		log.Error("subscription update failed", zap.String("reason", "missing_deps"))
		return dto.CreateSubscriptionResult{}, wrapError.ErrSubscriptionUpdateFailed
	}

	sub, plan, err := s.loadUpdateTargets(log, payload)
	if err != nil {
		return dto.CreateSubscriptionResult{}, err
	}
	if !isFreeTrialPlan(plan) && !isAllowedBillingCycle(payload.BillingCycle) {
		log.Warn("subscription update failed",
			zap.String("subscription_id", sub.ID),
			zap.String("plan_id", plan.ID),
			zap.Int("billing_cycle", payload.BillingCycle),
			zap.String("reason", "invalid_billing_cycle"),
		)
		return dto.CreateSubscriptionResult{}, wrapError.ErrInvalidRequest
	}
	return s.persistPlanChange(log, ctx, sub, plan, payload)
}

func (s *SubscriptionService) loadUpdateTargets(log *zap.Logger, payload dto.UpdateSubscriptionPayload) (Subscription, plans.Plan, error) {
	sub, err := s.Repo.GetByID(log, strings.TrimSpace(payload.SubscriptionID))
	if err != nil {
		return Subscription{}, plans.Plan{}, subscriptionUpdateLookupError(log, payload.SubscriptionID, err)
	}
	plan, err := s.Plan.GetByID(log, strings.TrimSpace(payload.PlanID))
	if err != nil {
		return Subscription{}, plans.Plan{}, planLookupError(log, payload.PlanID, err)
	}
	return sub, plan, nil
}

func subscriptionUpdateLookupError(log *zap.Logger, subscriptionID string, err error) error {
	if errors.Is(err, ErrSubscriptionNotFound) {
		log.Warn("subscription update failed",
			zap.String("subscription_id", subscriptionID),
			zap.String("reason", "not_found"),
		)
		return wrapError.ErrSubscriptionNotFound
	}
	log.Error("subscription update failed",
		zap.String("subscription_id", subscriptionID),
		zap.String("reason", "lookup"),
		zap.Error(err),
	)
	return wrapError.ErrSubscriptionUpdateFailed
}

func planLookupError(log *zap.Logger, planID string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn("subscription update failed",
			zap.String("plan_id", planID),
			zap.String("reason", "plan_not_found"),
		)
		return wrapError.ErrPlanNotFound
	}
	log.Error("subscription update failed",
		zap.String("plan_id", planID),
		zap.String("reason", "plan_lookup"),
		zap.Error(err),
	)
	return wrapError.ErrSubscriptionUpdateFailed
}

func (s *SubscriptionService) persistPlanChange(log *zap.Logger, ctx context.Context, sub Subscription, plan plans.Plan, payload dto.UpdateSubscriptionPayload) (dto.CreateSubscriptionResult, error) {
	change := buildSubscriptionChange(sub, plan, payload, time.Now())
	if change.Trial || change.Result.Price <= 0 {
		if err := s.saveSubscriptionUpdates(log, sub.ID, change.Updates); err != nil {
			return dto.CreateSubscriptionResult{}, err
		}
		if err := s.activateTenant(log, sub.TenantID); err != nil {
			return dto.CreateSubscriptionResult{}, err
		}
		s.logSubscriptionUpdate(log, sub, plan, change.Result)
		return change.Result, nil
	}

	orderID, err := s.openPaidSubscription(log, ctx, paidSubscription{
		Sub:            sub,
		HospitalName:   change.Result.HospitalName,
		CustomerID:     payload.CustomerID,
		OrganisationID: payload.OrganisationID,
		Price:          change.Result.Price,
		Renewal:        true,
	})
	if err != nil {
		return dto.CreateSubscriptionResult{}, err
	}
	change.Result.OrderID = orderID
	change.Result.KeyID = s.KeyID
	if err := s.saveSubscriptionUpdates(log, sub.ID, change.Updates); err != nil {
		return dto.CreateSubscriptionResult{}, err
	}
	s.logSubscriptionUpdate(log, sub, plan, change.Result)
	return change.Result, nil
}

func (s *SubscriptionService) logSubscriptionUpdate(log *zap.Logger, sub Subscription, plan plans.Plan, result dto.CreateSubscriptionResult) {
	log.Info("subscription update success",
		zap.String("subscription_id", sub.ID),
		zap.String("tenant_id", sub.TenantID),
		zap.String("plan_id", plan.ID),
		zap.String("status", result.Status),
		zap.String("billing_cycle", result.BillingCycle),
		zap.Float64("price", result.Price),
		zap.String("order_id", result.OrderID),
	)
}

func (s *SubscriptionService) createSubscriptionOrder(log *zap.Logger, ctx context.Context, sub Subscription, hospitalName string, price float64) (string, error) {
	if s.Orders == nil {
		log.Error("subscription update failed",
			zap.String("subscription_id", sub.ID),
			zap.String("reason", "order_creator_missing"),
		)
		return "", wrapError.ErrOrderCreateFailed
	}
	amountPaise := payments.AmountInPaise(price)
	orderResp, err := s.Orders.CreateOrder(log, ctx, paymentdto.CreateOrderRequest{
		Amount:   amountPaise,
		Currency: payments.CurrencyINR,
		Receipt:  truncateReceipt(sub.ID),
		Notes: map[string]string{
			"subscription_id": sub.ID,
			"tenant_id":       sub.TenantID,
			"hospital_name":   hospitalName,
		},
	})
	if err != nil {
		log.Error("subscription update order failed",
			zap.String("subscription_id", sub.ID),
			zap.Int64("amount_paise", amountPaise),
			zap.String("currency", payments.CurrencyINR),
			zap.Error(err),
		)
		return "", err
	}
	return orderResp.ID, nil
}

func (s *SubscriptionService) saveSubscriptionUpdates(log *zap.Logger, subscriptionID string, updates map[string]interface{}) error {
	if err := s.Repo.UpdateByID(log, nil, subscriptionID, updates); err != nil {
		log.Error("subscription update failed",
			zap.String("subscription_id", subscriptionID),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrSubscriptionUpdateFailed
	}
	return nil
}

type subscriptionChange struct {
	Updates map[string]interface{}
	Result  dto.CreateSubscriptionResult
	Trial   bool
}

func buildSubscriptionChange(sub Subscription, plan plans.Plan, payload dto.UpdateSubscriptionPayload, now time.Time) subscriptionChange {
	start := now
	change := subscriptionChange{
		Updates: map[string]interface{}{
			"plan_id":  plan.ID,
			"start_at": start,
		},
		Result: dto.CreateSubscriptionResult{
			SubscriptionID: sub.ID,
			TenantID:       sub.TenantID,
			HospitalName:   strings.TrimSpace(payload.HospitalName),
			StartAt:        &start,
		},
	}
	if isFreeTrialPlan(plan) {
		return applyFreeTrialChange(change, now)
	}
	return applyStandardChange(change, plan, payload, now)
}

func applyFreeTrialChange(change subscriptionChange, now time.Time) subscriptionChange {
	end := now.AddDate(0, 0, freeTrialDurationDays)
	change.Trial = true
	change.Result.Price = 0
	change.Result.BillingCycle = billingCycleTrial
	change.Result.Status = statusActive
	change.Result.EndAt = &end
	change.Updates["status"] = statusActive
	change.Updates["billing_cycle"] = billingCycleTrial
	change.Updates["price"] = float64(0)
	change.Updates["end_at"] = end
	return change
}

func applyStandardChange(change subscriptionChange, plan plans.Plan, payload dto.UpdateSubscriptionPayload, now time.Time) subscriptionChange {
	cycle := fmt.Sprintf("%d", payload.BillingCycle)
	end := subscriptionEndAt(now, cycle)
	price := totalSubscriptionPrice(plan, payload.Price, payload.BillingCycle)
	change.Result.Price = price
	change.Result.BillingCycle = cycle
	change.Result.Status = statusPendingPayment
	change.Result.EndAt = &end
	change.Updates["status"] = statusPendingPayment
	change.Updates["billing_cycle"] = cycle
	change.Updates["price"] = price
	change.Updates["end_at"] = end
	return change
}

// ActivateBySubscriptionID marks a pending_payment subscription active after a verified
// Razorpay webhook (order.paid), then activates the tenant. Idempotent when already active.
// Payment ids stay on the payment row; this only changes the subscription period.
func (s *SubscriptionService) ActivateBySubscriptionID(log *zap.Logger, subscriptionID, providerPaymentID string) error {
	log = ensureLog(log)
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return wrapError.ErrInvalidRequest
	}

	sub, err := s.Repo.GetByID(log, subscriptionID)
	if err != nil {
		return activateLookupError(log, subscriptionID, err)
	}
	if sub.Status == statusActive {
		if err := s.activateTenant(log, sub.TenantID); err != nil {
			return err
		}
		log.Info("subscription activate skipped",
			zap.String("subscription_id", sub.ID),
			zap.String("reason", "already_active"),
		)
		return nil
	}

	now := time.Now()
	if err := s.Repo.UpdateByID(log, nil, sub.ID, map[string]interface{}{
		"status":   statusActive,
		"start_at": now,
		"end_at":   subscriptionEndAt(now, sub.BillingCycle),
	}); err != nil {
		log.Error("subscription activate failed",
			zap.String("subscription_id", sub.ID),
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
		zap.String("provider_payment_id", providerPaymentID),
	)
	return nil
}

func activateLookupError(log *zap.Logger, subscriptionID string, err error) error {
	if errors.Is(err, ErrSubscriptionNotFound) {
		log.Warn("subscription activate failed",
			zap.String("subscription_id", subscriptionID),
			zap.String("reason", "not_found"),
		)
		return wrapError.ErrSubscriptionNotFound
	}
	log.Error("subscription activate failed",
		zap.String("subscription_id", subscriptionID),
		zap.String("reason", "lookup"),
		zap.Error(err),
	)
	return wrapError.ErrSubscriptionUpdateFailed
}

// ConfirmCheckout stores Razorpay Checkout success fields on the payment.
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
	if s.Billing == nil || s.Repo == nil {
		log.Error("subscription checkout store failed", zap.String("reason", "missing_deps"))
		return dto.ConfirmCheckoutResult{}, wrapError.ErrSubscriptionUpdateFailed
	}

	confirmed, err := s.Billing.ConfirmSubscriptionPayment(log, context.Background(), paymentdto.ConfirmSubscriptionPaymentRequest{
		OrderID:           payload.RazorpayOrderID,
		ProviderPaymentID: payload.RazorpayPaymentID,
		PaymentSignature:  payload.RazorpaySignature,
	})
	if err != nil {
		log.Error("subscription checkout store failed",
			zap.String("order_id", payload.RazorpayOrderID),
			zap.String("reason", "payment_api"),
			zap.Error(err),
		)
		return dto.ConfirmCheckoutResult{}, err
	}

	sub, err := s.Repo.GetByID(log, confirmed.SubscriptionID)
	if err != nil {
		return dto.ConfirmCheckoutResult{}, activateLookupError(log, confirmed.SubscriptionID, err)
	}
	log.Info("subscription checkout store success",
		zap.String("subscription_id", sub.ID),
		zap.String("order_id", payload.RazorpayOrderID),
		zap.String("payment_id", confirmed.PaymentID),
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
	if s.Repo == nil {
		log.Error("subscription end check failed", zap.String("reason", "repo_missing"))
		return dto.CheckEndResult{}, wrapError.ErrOrganisationFetchFailed
	}

	sub, err := s.Repo.GetLatestByTenantID(log, tenantID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			log.Warn("subscription end check failed",
				zap.String("tenant_id", tenantID),
				zap.String("reason", "subscription_not_found"),
			)
			return dto.CheckEndResult{}, wrapError.ErrSubscriptionNotFound
		}
		log.Error("subscription end check failed",
			zap.String("reason", "lookup"),
			zap.Error(err),
		)
		return dto.CheckEndResult{}, wrapError.ErrSubscriptionUpdateFailed
	}

	result := periodEndResult(organisationID, sub, time.Now())
	log.Info("subscription end check success",
		zap.String("subscription_id", sub.ID),
		zap.Bool("ended", result.Ended),
		zap.String("message", result.Message),
	)
	return result, nil
}

// ExpiredByTenantID reports whether the tenant's latest subscription has reached end_at.
// A tenant with no subscription is not expired.
func (s *SubscriptionService) ExpiredByTenantID(log *zap.Logger, tenantID string) (bool, error) {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return false, nil
	}
	sub, err := s.Repo.GetLatestByTenantID(log, tenantID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return false, nil
		}
		log.Error("subscription end check failed",
			zap.String("tenant_id", tenantID),
			zap.String("reason", "lookup"),
			zap.Error(err),
		)
		return false, err
	}
	return periodEndResult("", sub, time.Now()).Ended, nil
}

func (s *SubscriptionService) GetByTenantID(log *zap.Logger, tenantID string) (dto.SubscriptionDetail, error) {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		log.Warn("subscription lookup failed", zap.String("reason", "validation"))
		return dto.SubscriptionDetail{}, wrapError.ErrInvalidRequest
	}
	if s.Repo == nil {
		log.Error("subscription lookup failed", zap.String("reason", "repo_missing"))
		return dto.SubscriptionDetail{}, wrapError.ErrSubscriptionFetchFailed
	}

	row, err := s.Repo.GetLatestWithPlanByTenantID(log, tenantID)
	if err != nil {
		return dto.SubscriptionDetail{}, subscriptionLookupError(log, tenantID, err)
	}

	log.Info("subscription lookup success",
		zap.String("subscription_id", row.ID),
		zap.String("tenant_id", tenantID),
		zap.String("plan_id", row.PlanID),
	)
	detail := toSubscriptionDetail(row, time.Now())
	detail.OrderID = s.latestOrderID(log, row.ID)
	return detail, nil
}

func subscriptionLookupError(log *zap.Logger, tenantID string, err error) error {
	if errors.Is(err, ErrSubscriptionNotFound) {
		log.Warn("subscription lookup failed",
			zap.String("tenant_id", tenantID),
			zap.String("reason", "not_found"),
		)
		return wrapError.ErrSubscriptionNotFound
	}
	log.Error("subscription lookup failed",
		zap.String("tenant_id", tenantID),
		zap.String("reason", "db_read"),
		zap.Error(err),
	)
	return wrapError.ErrSubscriptionFetchFailed
}

func toSubscriptionDetail(row SubscriptionPlanRow, now time.Time) dto.SubscriptionDetail {
	details := map[string]any(row.PlanDetails)
	if details == nil {
		details = map[string]any{}
	}
	return dto.SubscriptionDetail{
		SubscriptionID: row.ID,
		TenantID:       row.TenantID,
		PlanID:         row.PlanID,
		Status:         row.Status,
		BillingCycle:   row.BillingCycle,
		Price:          row.Price,
		StartAt:        row.StartAt,
		EndAt:          row.EndAt,
		Expired:        periodEndResult("", Subscription{EndAt: row.EndAt, BillingCycle: row.BillingCycle}, now).Ended,
		Plan: dto.SubscriptionPlan{
			Name:         row.PlanName,
			Status:       row.PlanStatus,
			MonthlyPrice: row.MonthlyPrice,
			Discount:     row.Discount,
			PlanDetails:  details,
		},
	}
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
				zap.String("reason", "organisation_not_found"),
			)
			return "", wrapError.ErrOrganisationNotFound
		}
		log.Error("subscription end check failed",
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
