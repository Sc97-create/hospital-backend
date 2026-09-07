package payments

import (
	"encoding/json"
	invoiceDto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/providers"
	"hospital-backend/pkg/constants"
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
}

func NewWebhookService(
	db *gorm.DB,
	webhookRepository IWebhookRepository,
	paymentsService PaymentInvoiceLookup,
	paymentAttempts PaymentAttemptServicer,
	paymentFactory providers.IPaymentFactory,
	fulfillment IPaymentFulfillment,
	fulfillmentSvc FulfillmentServicer,
) *IWebhookService {
	return &IWebhookService{
		db:                db,
		WebhookRepository: webhookRepository,
		PaymentsService:   paymentsService,
		PaymentAttempts:   paymentAttempts,
		PaymentFactory:    paymentFactory,
		Fulfillment:       fulfillment,
		FulfillmentSvc:    fulfillmentSvc,
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
	)

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

func (w *IWebhookService) toWebhookEvent(dtowebhookevent dto.ParsedWebhookEvent, paymentAttemptID string) WebhookEvents {
	var webhookresponse datatypes.JSONMap
	err := json.Unmarshal(dtowebhookevent.RawPayload, &webhookresponse)
	if err != nil {
		return WebhookEvents{}
	}
	return WebhookEvents{
		ID:               uuid.NewString(),
		PaymentAttemptID: paymentAttemptID,
		EventType:        dtowebhookevent.EventType,
		ProviderResponse: webhookresponse,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
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
