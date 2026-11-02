package payments

import (
	"encoding/json"
	invoiceDto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/providers"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IWebhookService struct {
	db                *gorm.DB
	WebhookRepository IWebhookRepository
	PaymentsService   PaymentInvoiceLookup
	PaymentAttempts   PaymentAttemptServicer
	PaymentFactory    providers.IPaymentFactory
	Fulfillment       IPaymentFulfillment
	FulfillmentSvc    FulfillmentServicer
	TenantEntitlement TenantEntitlementActivator
}

// TenantEntitlementActivator is the common hook used after a webhook signature is verified
// to activate the tenant (and its pending subscription) for a paid Razorpay order.
// The subscription id comes from the invoice linked to the payment with that order_id.
type TenantEntitlementActivator interface {
	ActivateBySubscriptionID(log *zap.Logger, subscriptionID, providerPaymentID string) error
}

func NewWebhookService(
	db *gorm.DB,
	webhookRepository IWebhookRepository,
	paymentsService PaymentInvoiceLookup,
	paymentAttempts PaymentAttemptServicer,
	paymentFactory providers.IPaymentFactory,
	fulfillment IPaymentFulfillment,
	fulfillmentSvc FulfillmentServicer,
	tenantEntitlement TenantEntitlementActivator,
) *IWebhookService {
	return &IWebhookService{
		db:                db,
		WebhookRepository: webhookRepository,
		PaymentsService:   paymentsService,
		PaymentAttempts:   paymentAttempts,
		PaymentFactory:    paymentFactory,
		Fulfillment:       fulfillment,
		FulfillmentSvc:    fulfillmentSvc,
		TenantEntitlement: tenantEntitlement,
	}
}

// ProcessWebhook verifies, claims, and handles a provider webhook.
// Returns (verified, err). verified=false → reject (401). verified=true + nil err → ack (200).
func (w *IWebhookService) ProcessWebhook(log *zap.Logger, payload []byte, signature string, provider string) (bool, error) {
	log = ensureLog(log)
	gateway, err := w.PaymentFactory.GetProvider(provider)
	if err != nil {
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("reason", "provider"),
			zap.Error(err),
		)
		return false, err
	}
	isverified, err := gateway.VerifySignature(payload, signature)
	if err != nil || !isverified {
		reason := "signature_invalid"
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "missing") {
			reason = "missing_signature"
		}
		log.Warn("payment webhook rejected",
			zap.String("provider", provider),
			zap.String("reason", reason),
		)
		return false, err
	}
	dtowebhookevent, err := gateway.ParseWebhookEvent(payload)
	if err != nil {
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("reason", "parse"),
			zap.Error(err),
		)
		return false, err
	}
	log.Info("payment webhook event",
		zap.String("provider", provider),
		zap.String("event_type", dtowebhookevent.EventType),
		zap.String("provider_link_id", dtowebhookevent.ProviderLinkID),
		zap.String("provider_order_id", dtowebhookevent.ProviderOrderID),
	)

	if dtowebhookevent.EventType == constants.OrderPaid {
		return w.handleOrderPaid(log, provider, dtowebhookevent)
	}

	paymentAttempt, claimed, err := w.PaymentAttempts.ClaimForProcessing(log, dtowebhookevent.ProviderLinkID)
	if err != nil {
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("provider_link_id", dtowebhookevent.ProviderLinkID),
			zap.String("reason", "claim"),
			zap.Error(err),
		)
		return false, err
	}
	if !claimed {
		log.Warn("payment webhook duplicate skipped",
			zap.String("provider", provider),
			zap.String("provider_link_id", dtowebhookevent.ProviderLinkID),
			zap.String("reason", "already_claimed"),
		)
		return true, nil
	}

	webhookEvent := w.toWebhookEvent(dtowebhookevent, paymentAttempt.ID)
	err = w.WebhookRepository.CreateWebhookEvent(log, webhookEvent)
	if err != nil {
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("payment_attempt_id", paymentAttempt.ID),
			zap.String("reason", "persist_event"),
			zap.Error(err),
		)
		return false, err
	}

	begin := w.db.Begin()
	invoiceInfo, err := w.getInvoiceForUpdate(log, paymentAttempt.PaymentID)
	if err != nil {
		begin.Rollback()
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("payment_id", paymentAttempt.PaymentID),
			zap.String("reason", "load_payment"),
			zap.Error(err),
		)
		return false, err
	}
	medicineInventoryDet, err := w.getMedInventoryForUpdate(invoiceInfo.InvoiceID)
	if err != nil {
		begin.Rollback()
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("invoice_id", invoiceInfo.InvoiceID),
			zap.String("reason", "inventory"),
			zap.Error(err),
		)
		return false, err
	}
	var prescriptionID string
	if len(medicineInventoryDet) > 0 {
		prescriptionID = medicineInventoryDet[0].PrescriptionID
	}

	switch dtowebhookevent.EventType {
	case constants.PaymentLinkPaid:
		if dtowebhookevent.AcceptPartial && dtowebhookevent.AmountPaid < invoiceInfo.Amount {
			paymentAttempt.PaymentStatus = constants.StatusPartiallyPaid
			paymentAttempt.PaymentLinkStatus = constants.StatusPartiallyPaid
			paymentAttempt.AmountPaid = dtowebhookevent.AmountPaid
			err = w.PaymentAttempts.UpdatePaymentAttemptStatus(log, begin, paymentAttempt)
			if err != nil {
				begin.Rollback()
				log.Error("payment webhook failed",
					zap.String("invoice_id", invoiceInfo.InvoiceID),
					zap.String("reason", "partial_status"),
					zap.Error(err),
				)
				return false, err
			}
			begin.Commit()
			log.Warn("payment webhook partial skipped",
				zap.String("invoice_id", invoiceInfo.InvoiceID),
				zap.String("payment_id", invoiceInfo.ID),
				zap.String("payment_attempt_id", paymentAttempt.ID),
				zap.String("reason", "partial_amount"),
			)
			return true, nil
		}
		err = w.updatePaymentAttempt(log, begin, paymentAttempt, dtowebhookevent)
		if err != nil {
			begin.Rollback()
			log.Error("payment webhook failed",
				zap.String("invoice_id", invoiceInfo.InvoiceID),
				zap.String("reason", "attempt_update"),
				zap.Error(err),
			)
			return false, err
		}
		err = w.FulfillmentSvc.FulfillPaidInvoice(log, begin, invoiceInfo.InvoiceID)
		if err != nil {
			begin.Rollback()
			log.Error("payment webhook failed",
				zap.String("invoice_id", invoiceInfo.InvoiceID),
				zap.String("reason", "fulfill"),
				zap.Error(err),
			)
			return false, err
		}
		begin.Commit()
		log.Info("payment webhook paid success",
			zap.String("provider", provider),
			zap.String("event_type", dtowebhookevent.EventType),
			zap.String("invoice_id", invoiceInfo.InvoiceID),
			zap.String("payment_id", invoiceInfo.ID),
			zap.String("prescription_id", prescriptionID),
			zap.String("payment_attempt_id", paymentAttempt.ID),
		)
		notifyErr := w.FulfillmentSvc.NotifyPaymentReceived(log, invoiceInfo, paymentAttempt.PaidAt)
		if notifyErr != nil {
			log.Error("payment notification failed",
				zap.String("invoice_id", invoiceInfo.InvoiceID),
				zap.String("payment_id", invoiceInfo.ID),
				zap.String("reason", "notification"),
				zap.Error(notifyErr),
			)
		} else {
			log.Info("payment notification enqueued",
				zap.String("invoice_id", invoiceInfo.InvoiceID),
				zap.Bool("notification_enqueued", true),
			)
		}
		return true, nil

	case constants.PaymentLinkCancelled:
		paymentAttempt.PaymentStatus = constants.StatusCancelled
		paymentAttempt.PaymentLinkStatus = constants.StatusCancelled
		err = w.PaymentAttempts.UpdatePaymentAttemptStatus(log, begin, paymentAttempt)
		if err != nil {
			begin.Rollback()
			log.Error("payment webhook failed",
				zap.String("provider_link_id", dtowebhookevent.ProviderLinkID),
				zap.String("reason", "cancel_status"),
				zap.Error(err),
			)
			return false, err
		}
		if prescriptionID != "" {
			err = w.Fulfillment.UpdateExtPrescriptionStatus(begin, prescriptionID, constants.StatusPaymentPending)
			if err != nil {
				begin.Rollback()
				log.Error("payment webhook failed",
					zap.String("prescription_id", prescriptionID),
					zap.String("reason", "prescription_status"),
					zap.Error(err),
				)
				return false, err
			}
		}
		begin.Commit()
		log.Warn("payment webhook link closed",
			zap.String("event_type", dtowebhookevent.EventType),
			zap.String("provider_link_id", dtowebhookevent.ProviderLinkID),
			zap.String("status", constants.StatusCancelled),
		)
		return true, nil

	case constants.PaymentLinkExpired:
		paymentAttempt.PaymentStatus = constants.StatusExpired
		paymentAttempt.PaymentLinkStatus = constants.StatusExpired
		err = w.PaymentAttempts.UpdatePaymentAttemptStatus(log, begin, paymentAttempt)
		if err != nil {
			begin.Rollback()
			log.Error("payment webhook failed",
				zap.String("provider_link_id", dtowebhookevent.ProviderLinkID),
				zap.String("reason", "expire_status"),
				zap.Error(err),
			)
			return false, err
		}
		if prescriptionID != "" {
			err = w.Fulfillment.UpdateExtPrescriptionStatus(begin, prescriptionID, constants.StatusPaymentPending)
			if err != nil {
				begin.Rollback()
				log.Error("payment webhook failed",
					zap.String("prescription_id", prescriptionID),
					zap.String("reason", "prescription_status"),
					zap.Error(err),
				)
				return false, err
			}
		}
		begin.Commit()
		log.Warn("payment webhook link closed",
			zap.String("event_type", dtowebhookevent.EventType),
			zap.String("provider_link_id", dtowebhookevent.ProviderLinkID),
			zap.String("status", constants.StatusExpired),
		)
		return true, nil
	}

	begin.Rollback()
	log.Warn("payment webhook unhandled event",
		zap.String("provider", provider),
		zap.String("event_type", dtowebhookevent.EventType),
		zap.String("provider_link_id", dtowebhookevent.ProviderLinkID),
		zap.String("reason", "unhandled_event_type"),
	)
	return true, nil
}

func (w *IWebhookService) handleOrderPaid(log *zap.Logger, provider string, event dto.ParsedWebhookEvent) (bool, error) {
	orderID := strings.TrimSpace(event.ProviderOrderID)
	if orderID == "" {
		log.Warn("payment webhook order.paid skipped",
			zap.String("provider", provider),
			zap.String("reason", "missing_order_id"),
		)
		return true, nil
	}

	webhookEvent := w.toWebhookEvent(event, "")
	if err := w.WebhookRepository.CreateWebhookEvent(log, webhookEvent); err != nil {
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("provider_order_id", orderID),
			zap.String("reason", "persist_event"),
			zap.Error(err),
		)
		return false, err
	}

	if w.TenantEntitlement == nil {
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("provider_order_id", orderID),
			zap.String("reason", "tenant_entitlement_activator_missing"),
		)
		return false, wrapError.ErrWebhookProcessFailed
	}

	subscriptionID, err := w.subscriptionIDByOrder(log, orderID, event.ProviderPaymentID)
	if err != nil {
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("provider_order_id", orderID),
			zap.String("reason", "subscription_lookup"),
			zap.Error(err),
		)
		return false, err
	}

	if err := w.TenantEntitlement.ActivateBySubscriptionID(log, subscriptionID, event.ProviderPaymentID); err != nil {
		log.Error("payment webhook failed",
			zap.String("provider", provider),
			zap.String("provider_order_id", orderID),
			zap.String("provider_payment_id", event.ProviderPaymentID),
			zap.String("reason", "activate_tenant_entitlement"),
			zap.Error(err),
		)
		return false, err
	}

	log.Info("payment webhook order.paid success",
		zap.String("provider", provider),
		zap.String("provider_order_id", orderID),
		zap.String("provider_payment_id", event.ProviderPaymentID),
	)
	return true, nil
}

func (w *IWebhookService) subscriptionIDByOrder(log *zap.Logger, orderID, providerPaymentID string) (string, error) {
	if w.db == nil {
		return "", wrapError.ErrWebhookProcessFailed
	}
	var row struct {
		SubscriptionID string `gorm:"column:subscription_id"`
	}
	err := w.db.Raw(`
		SELECT i.billing_id AS subscription_id
		FROM payments p
		JOIN invoices i ON i.id = p.invoice_id
		WHERE p.order_id = ?
		  AND i.payment_type IN ('subscription_new', 'subscription_renewal')
		LIMIT 1
	`, orderID).Scan(&row).Error
	if err != nil {
		return "", err
	}
	if row.SubscriptionID == "" {
		return "", wrapError.ErrSubscriptionNotFound
	}
	now := time.Now()
	if err := w.db.Model(&Payments{}).Where("order_id = ?", orderID).Updates(map[string]any{
		"provider_payment_id": providerPaymentID,
		"updated_at":          now,
	}).Error; err != nil {
		log.Error("payment webhook failed",
			zap.String("provider_order_id", orderID),
			zap.String("reason", "payment_update"),
			zap.Error(err),
		)
		return "", wrapError.ErrWebhookProcessFailed
	}
	if err := w.db.Model(&PaymentAttempts{}).Where("provider_order_id = ?", orderID).Updates(map[string]any{
		"provider_payment_id": providerPaymentID,
	}).Error; err != nil {
		log.Error("payment webhook failed",
			zap.String("provider_order_id", orderID),
			zap.String("reason", "attempt_update"),
			zap.Error(err),
		)
		return "", wrapError.ErrWebhookProcessFailed
	}
	return row.SubscriptionID, nil
}

func (w *IWebhookService) toWebhookEvent(dtowebhookevent dto.ParsedWebhookEvent, paymentAttemptID string) WebhookEvents {
	var webhookresponse datatypes.JSONMap
	err := json.Unmarshal(dtowebhookevent.RawPayload, &webhookresponse)
	if err != nil {
		return WebhookEvents{}
	}
	event := WebhookEvents{
		ID:               uuid.NewString(),
		EventType:        dtowebhookevent.EventType,
		ProviderResponse: webhookresponse,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if id := strings.TrimSpace(paymentAttemptID); id != "" {
		event.PaymentAttemptID = &id
	}
	return event
}

func (w *IWebhookService) updatePaymentAttempt(log *zap.Logger, tx *gorm.DB, paymentAttempt PaymentAttempts, dtowebhookevent dto.ParsedWebhookEvent) error {
	paymentAttempt.PaymentStatus = dtowebhookevent.PaymentStatus
	paymentAttempt.PaymentLinkStatus = dtowebhookevent.PaymentLinkStatus
	paymentAttempt.AmountPaid = dtowebhookevent.AmountPaid
	paymentAttempt.AmountTransferred = dtowebhookevent.AmountTransferred
	paymentAttempt.PayerAccountType = dtowebhookevent.PayerAccountType
	paymentAttempt.PaymentVPA = dtowebhookevent.PayerVPA
	paymentAttempt.ProviderOrderID = dtowebhookevent.ProviderOrderID
	paymentAttempt.ProviderPaymentID = dtowebhookevent.ProviderPaymentID
	paymentAttempt.ProviderReferenceID = dtowebhookevent.ReferenceID
	paymentAttempt.PaidAt = dtowebhookevent.PaidAt
	paymentAttempt.AcceptPartial = dtowebhookevent.AcceptPartial
	return w.PaymentAttempts.UpdatePaymentAttempt(log, tx, paymentAttempt)
}

func (w *IWebhookService) getInvoiceForUpdate(log *zap.Logger, paymentID string) (Payments, error) {
	query := `SELECT * FROM payments WHERE id = ?`
	return w.PaymentsService.FindInvoiceByPaymentAttempt(log, query, paymentID)
}

func (w *IWebhookService) getMedInventoryForUpdate(invoiceID string) ([]invoiceDto.MedInvoiceItemResponse, error) {
	return w.Fulfillment.GetMedicineInventoryDetByInvoiceID(invoiceID)
}
