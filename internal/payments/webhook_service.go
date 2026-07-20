package payments

import (
	"encoding/json"
	invoiceDto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/providers"
	"hospital-backend/pkg/constants"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IWebhookService struct {
	db                *gorm.DB
	WebhookRepository IWebhookRepository
	PaymentsService   *PaymentsService
	PaymentAttempts   *SPaymentAttempts
	PaymentFactory    *providers.PaymentFactory
	Fulfillment       IPaymentFulfillment
	FulfillmentSvc    *FulfillmentService
}

func NewWebhookService(
	db *gorm.DB,
	webhookRepository IWebhookRepository,
	paymentsService *PaymentsService,
	paymentAttempts *SPaymentAttempts,
	paymentFactory *providers.PaymentFactory,
	fulfillment IPaymentFulfillment,
	fulfillmentSvc *FulfillmentService,
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
func (w *IWebhookService) ProcessWebhook(payload []byte, signature string, provider string) (bool, error) {
	gateway, err := w.PaymentFactory.GetProvider(provider)
	if err != nil {
		return false, err
	}
	isverified, err := gateway.VerifySignature(payload, signature)
	if err != nil || !isverified {
		return false, err
	}
	dtowebhookevent, err := gateway.ParseWebhookEvent(payload)
	if err != nil {
		return false, err
	}
	// atomically claim the attempt: pending → processing. Guards against duplicate webhooks.
	paymentAttempt, claimed, err := w.PaymentAttempts.ClaimForProcessing(dtowebhookevent.ProviderLinkID)
	if err != nil {
		return false, err
	}
	if !claimed {
		// duplicate webhook — already being processed by another goroutine, ack and exit
		return true, nil
	}
	//store in webhook
	webhookEvent := w.toWebhookEvent(dtowebhookevent, paymentAttempt.ID)
	err = w.WebhookRepository.CreateWebhookEvent(webhookEvent)
	if err != nil {
		return false, err
	}
	begin := w.db.Begin()
	invoiceInfo, err := w.getInvoiceForUpdate(paymentAttempt.PaymentID)
	if err != nil {
		return false, err
	}
	medicineInventoryDet, err := w.getMedInventoryForUpdate(invoiceInfo.InvoiceID)
	if err != nil {
		begin.Rollback()
		return false, err
	}
	var prescriptionID string
	if len(medicineInventoryDet) > 0 {
		prescriptionID = medicineInventoryDet[0].PrescriptionID
	}
	switch dtowebhookevent.EventType {
	case constants.PaymentLinkPaid:
		// guard: partial payment — do not dispense until full amount is paid
		if dtowebhookevent.AcceptPartial && dtowebhookevent.AmountPaid < invoiceInfo.Amount {
			paymentAttempt.PaymentStatus = constants.StatusPartiallyPaid
			paymentAttempt.PaymentLinkStatus = constants.StatusPartiallyPaid
			paymentAttempt.AmountPaid = dtowebhookevent.AmountPaid
			err = w.PaymentAttempts.UpdatePaymentAttemptStatus(begin, paymentAttempt)
			if err != nil {
				begin.Rollback()
				return false, err
			}
			begin.Commit()
			return true, nil // ack webhook but skip dispensing
		}
		// update payment attempt fields (status already set to processing by ClaimForProcessing)
		err = w.updatePaymentAttempt(begin, paymentAttempt, dtowebhookevent)
		if err != nil {
			begin.Rollback()
			return false, err
		}
		err = w.FulfillmentSvc.FulfillPaidInvoice(begin, invoiceInfo.InvoiceID)
		if err != nil {
			begin.Rollback()
			return false, err
		}
		begin.Commit()
		err = w.FulfillmentSvc.NotifyPaymentReceived(invoiceInfo, paymentAttempt.PaidAt)
		if err != nil {
			return false, err
		}
		return true, nil
	case constants.PaymentLinkCancelled:
		paymentAttempt.PaymentStatus = constants.StatusCancelled
		paymentAttempt.PaymentLinkStatus = constants.StatusCancelled
		err = w.PaymentAttempts.UpdatePaymentAttemptStatus(begin, paymentAttempt)
		if err != nil {
			begin.Rollback()
			return false, err
		}
		err = w.Fulfillment.UpdateInvoiceStatus(begin, invoiceInfo.InvoiceID, constants.StatusCancelled)
		if err != nil {
			begin.Rollback()
			return false, err
		}
		err = w.Fulfillment.UpdateExtPrescriptionStatus(begin, prescriptionID, constants.StatusCancelled)
		if err != nil {
			begin.Rollback()
			return false, err
		}
		begin.Commit()
		return true, nil
	case constants.PaymentLinkExpired:
		// invoice stays unpaid — cashier can generate a new payment link
		paymentAttempt.PaymentStatus = constants.StatusExpired
		paymentAttempt.PaymentLinkStatus = constants.StatusExpired
		err = w.PaymentAttempts.UpdatePaymentAttemptStatus(begin, paymentAttempt)
		if err != nil {
			begin.Rollback()
			return false, err
		}
		err = w.Fulfillment.UpdateInvoiceStatus(begin, invoiceInfo.InvoiceID, constants.StatusExpired)
		if err != nil {
			begin.Rollback()
			return false, err
		}
		err = w.Fulfillment.UpdateExtPrescriptionStatus(begin, prescriptionID, constants.StatusExpired)
		if err != nil {
			begin.Rollback()
			return false, err
		}
		begin.Commit()
		return true, nil
	}
	return false, nil
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
func (w *IWebhookService) updatePaymentAttempt(tx *gorm.DB, paymentAttempt PaymentAttempts, dtowebhookevent dto.ParsedWebhookEvent) error {
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
	err := w.PaymentAttempts.UpdatePaymentAttempt(tx, paymentAttempt)
	if err != nil {
		return err
	}
	return nil
}
func (w *IWebhookService) getInvoiceForUpdate(paymentID string) (Payments, error) {
	query := `SELECT * FROM payments WHERE id = ?`
	paymentInfo, err := w.PaymentsService.FindInvoiceByPaymentAttempt(query, paymentID)
	if err != nil {
		return Payments{}, err
	}
	return paymentInfo, nil
}
func (w *IWebhookService) getMedInventoryForUpdate(invoiceID string) ([]invoiceDto.MedInvoiceItemResponse, error) {

	medicineInventoryDet, err := w.Fulfillment.GetMedicineInventoryDetByInvoiceID(invoiceID)
	if err != nil {
		return nil, err
	}
	return medicineInventoryDet, nil
}
