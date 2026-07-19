package payments

import (
	"context"
	"encoding/json"
	invoiceDto "hospital-backend/internal/billing/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/providers"
	"hospital-backend/pkg/constants"
	"hospital-backend/pkg/types"
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
}

func NewWebhookService(db *gorm.DB, webhookRepository IWebhookRepository, paymentsService *PaymentsService, paymentAttempts *SPaymentAttempts, paymentFactory *providers.PaymentFactory, fulfillment IPaymentFulfillment) *IWebhookService {
	return &IWebhookService{db: db, WebhookRepository: webhookRepository, PaymentsService: paymentsService, PaymentAttempts: paymentAttempts, PaymentFactory: paymentFactory, Fulfillment: fulfillment}
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

		var bulkMedicineMvmt []types.MedicineStockMovements

		allFullyDispensed := true // XOR flag: flips false if any item is partially dispensed
		for _, each := range medicineInventoryDet {
			// build stock movement record for bulk insert
			var eachMedicineMvmt types.MedicineStockMovements
			eachMedicineMvmt.ID = uuid.NewString()
			eachMedicineMvmt.MedicineID = each.MedicineID
			eachMedicineMvmt.MedicineInventoryID = each.MedicineInventoryID
			eachMedicineMvmt.OrganisationID = each.OrganisationID
			eachMedicineMvmt.MovementType = types.Dispense
			eachMedicineMvmt.QtyChanged = int(each.DispensedQty)
			eachMedicineMvmt.SourceType = types.PatientMedicineOrder
			eachMedicineMvmt.UnitPriceAtTimeOfMvmt = each.Pricing.UnitPrice
			eachMedicineMvmt.BalanceAfterMvmt = int(each.CurrentStockUnitAfterDispense)
			eachMedicineMvmt.CreatedBy = each.CashierID
			bulkMedicineMvmt = append(bulkMedicineMvmt, eachMedicineMvmt)

			// atomically decrement medicine inventory stock by dispensed qty
			err = w.Fulfillment.UpdateMedInventoryStock(begin, each.MedicineInventoryID, each.DispensedQty)
			if err != nil {
				begin.Rollback()
				return false, err
			}

			// increment balance_after_dispense on prescription item
			err = w.Fulfillment.UpdateDispenseItemQty(begin, each.PrescriptionItemID, each.DispensedQty)
			if err != nil {
				begin.Rollback()
				return false, err
			}
			// item-level status: fully or partially dispensed
			newBalance := each.AlreadyDispensed + each.DispensedQty
			if newBalance >= each.PrescribedQty {
				err = w.Fulfillment.UpdateIPrescriptionStatus(begin, each.PrescriptionItemID, constants.StatusFullyDispensed)
			} else {
				allFullyDispensed = false // at least one item still pending
				err = w.Fulfillment.UpdateIPrescriptionStatus(begin, each.PrescriptionItemID, constants.StatusPartiallyDispensed)
			} // item-level status updated
			if err != nil {
				begin.Rollback()
				return false, err
			}
		}

		// bulk insert all stock movement records
		err = w.Fulfillment.CreateMedicineMvmt(begin, bulkMedicineMvmt)
		if err != nil {
			begin.Rollback()
			return false, err
		}

		// prescription-level status derived from XOR of all item statuses
		prescStatus := constants.StatusPartiallyDispensed
		if allFullyDispensed {
			prescStatus = constants.StatusFullyDispensed
		}
		if prescriptionID != "" {
			err = w.Fulfillment.UpdateExtPrescriptionStatus(begin, prescriptionID, prescStatus)
			if err != nil {
				begin.Rollback()
				return false, err
			}
		}
		err = w.Fulfillment.UpdateInvoiceStatus(begin, invoiceInfo.InvoiceID, constants.InvoicePaid)
		if err != nil {
			begin.Rollback()
			return false, err
		} //invoice status updated
		begin.Commit()
		ctx := context.Background()
		notificationRequest, err := w.createNotification(invoiceInfo, constants.PaymentReceivedEvent, paymentAttempt.PaidAt)
		if err != nil {
			return false, err
		}
		err = w.Fulfillment.CreateNotification(ctx, notificationRequest)
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
func (w *IWebhookService) createNotification(invoiceInfo Payments, eventType string, paidAt time.Time) (notificationdto.CreateRequest, error) {
	patientInfo, err := w.Fulfillment.GetNotificationPatientByID(invoiceInfo.PatientID)
	if err != nil {
		return notificationdto.CreateRequest{}, err
	}
	patientInfo["amount_paid"] = invoiceInfo.Amount
	patientInfo["payment_status"] = eventType
	patientInfo["currency"] = invoiceInfo.Currency
	patientInfo["paid_at"] = paidAt.Format("02 Jan 2006 15:04:05")
	var notificationRequest notificationdto.CreateRequest
	notificationRequest.Data = patientInfo
	notificationRequest.NotificationType = constants.PaymentReceivedEvent
	notificationRequest.Subject = constants.PaymentReceivedSubject
	return notificationRequest, nil
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
