package payments

import (
	"context"
	invoiceDto "hospital-backend/internal/billing/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/pkg/constants"
	"hospital-backend/pkg/types"
	wrapError "hospital-backend/shared/error"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FulfillmentService orchestrates post-payment side effects (stock, movements, prescription, invoice).
// Shared by webhook paid and cash/qr manual confirm.
type FulfillmentService struct {
	Deps IPaymentFulfillment
}

func NewFulfillmentService(deps IPaymentFulfillment) *FulfillmentService {
	return &FulfillmentService{Deps: deps}
}

// FulfillPaidInvoice updates inventory, stock movements, prescription status, and marks the invoice paid.
// Caller owns the transaction (commit/rollback).
func (s *FulfillmentService) FulfillPaidInvoice(log *zap.Logger, tx *gorm.DB, invoiceID string) error {
	log = ensureLog(log)
	medicineInventoryDet, err := s.Deps.GetMedicineInventoryDetByInvoiceID(invoiceID)
	if err != nil {
		log.Error("payment fulfillment failed",
			zap.String("invoice_id", invoiceID),
			zap.String("reason", "inventory"),
			zap.Error(err),
		)
		return wrapError.ErrPaymentFulfillFailed
	}
	var prescriptionID string
	if len(medicineInventoryDet) > 0 {
		prescriptionID = medicineInventoryDet[0].PrescriptionID
	}

	var bulkMedicineMvmt []types.MedicineStockMovements
	remHashing := make(map[string]int64)
	for i, each := range medicineInventoryDet {
		//if prescriptionid is same then remaining becomes global and can be reset once new prescription
		if each.DispensedQty > 0 {
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

			err = s.Deps.UpdateMedInventoryStock(tx, each.MedicineInventoryID, each.DispensedQty)
			if err != nil {
				log.Error("payment fulfillment failed",
					zap.String("invoice_id", invoiceID),
					zap.String("reason", "stock"),
					zap.Error(err),
				)
				return wrapError.ErrPaymentFulfillFailed
			}
		}

		err = s.Deps.UpdateDispenseItemQty(tx, each.PrescriptionItemID, each.DispensedQty)
		if err != nil {
			log.Error("payment fulfillment failed",
				zap.String("invoice_id", invoiceID),
				zap.String("reason", "dispense"),
				zap.Error(err),
			)
			return wrapError.ErrPaymentFulfillFailed
		}

		// Same prescription item can appear on multiple invoice lines (different batches).
		// Track running remaining across those lines so the final item status is correct.
		remaining := int64(0)
		if value, ok := remHashing[each.PrescriptionItemID]; ok {
			remaining = value - each.DispensedQty
		} else {
			remaining = each.PrescribedQty - each.DispensedQty
		}
		remHashing[each.PrescriptionItemID] = remaining

		outOfStock := false
		if remaining != 0 {
			medicineInventoryDet[i].PrescriptionItemStatus = constants.StatusPartiallyDispensed
			// Took all available stock and batch is empty, but prescribed qty still remains → OOS.
			// If patient demanded less than current stock (or skipped while stock exists) → tentative only, oos=false.
			if each.DispensedQty > 0 && each.DispensedQty >= each.CurrentStockUnit && each.CurrentStockUnitAfterDispense == 0 {
				outOfStock = true
			}
			err = s.Deps.UpdateIPrescriptionStatus(tx, each.PrescriptionItemID, constants.StatusPartiallyDispensed, outOfStock)
		} else {
			medicineInventoryDet[i].PrescriptionItemStatus = constants.StatusFullyDispensed
			err = s.Deps.UpdateIPrescriptionStatus(tx, each.PrescriptionItemID, constants.StatusFullyDispensed, false)
		}
		if err != nil {
			log.Error("payment fulfillment failed",
				zap.String("invoice_id", invoiceID),
				zap.String("reason", "item_status"),
				zap.Error(err),
			)
			return wrapError.ErrPaymentFulfillFailed
		}
	}

	if len(bulkMedicineMvmt) > 0 {
		err = s.Deps.CreateMedicineMvmt(tx, bulkMedicineMvmt)
		if err != nil {
			log.Error("payment fulfillment failed",
				zap.String("invoice_id", invoiceID),
				zap.String("reason", "stock_mvmt"),
				zap.Error(err),
			)
			return wrapError.ErrPaymentFulfillFailed
		}
	}

	if prescriptionID != "" {
		// Parent resolve must use one final status per prescription_item_id.
		// Multi-batch lines leave earlier slice entries as partially_dispensed.
		parentItems := finalPrescriptionItemStatuses(medicineInventoryDet)
		err = s.Deps.ResolveAndUpdateParentPrescriptionStatus(tx, prescriptionID, parentItems)
		if err != nil {
			log.Error("payment fulfillment failed",
				zap.String("invoice_id", invoiceID),
				zap.String("prescription_id", prescriptionID),
				zap.String("reason", "parent_status"),
				zap.Error(err),
			)
			return wrapError.ErrPaymentFulfillFailed
		}
	}

	if err = s.Deps.UpdateInvoiceStatus(tx, invoiceID, constants.InvoicePaid); err != nil {
		log.Error("payment fulfillment failed",
			zap.String("invoice_id", invoiceID),
			zap.String("reason", "invoice_status"),
			zap.Error(err),
		)
		return wrapError.ErrPaymentFulfillFailed
	}

	log.Info("payment fulfillment success",
		zap.String("invoice_id", invoiceID),
		zap.String("prescription_id", prescriptionID),
		zap.Int("item_count", len(medicineInventoryDet)),
		zap.Int("movement_count", len(bulkMedicineMvmt)),
	)
	return nil
}

func (s *FulfillmentService) NotifyPaymentReceived(log *zap.Logger, payment Payments, paidAt time.Time) error {
	log = ensureLog(log)
	patientInfo, err := s.Deps.GetNotificationPatientByID(payment.PatientID)
	if err != nil {
		return err
	}
	patientInfo["amount_paid"] = payment.Amount
	patientInfo["payment_status"] = constants.PaymentReceivedEvent
	patientInfo["currency"] = payment.Currency
	patientInfo["paid_at"] = paidAt.Format("02 Jan 2006 15:04:05")

	return s.Deps.CreateNotification(context.Background(), notificationdto.CreateRequest{
		Data:             patientInfo,
		NotificationType: constants.PaymentReceivedEvent,
		Subject:          constants.PaymentReceivedSubject,
	})
}

// finalPrescriptionItemStatuses collapses multi-batch invoice lines to one entry per
// prescription_item_id, keeping the last computed status (after remHashing).
func finalPrescriptionItemStatuses(items []invoiceDto.MedInvoiceItemResponse) []invoiceDto.MedInvoiceItemResponse {
	indexByItem := make(map[string]int, len(items))
	out := make([]invoiceDto.MedInvoiceItemResponse, 0, len(items))
	for _, each := range items {
		if idx, ok := indexByItem[each.PrescriptionItemID]; ok {
			out[idx] = each
			continue
		}
		indexByItem[each.PrescriptionItemID] = len(out)
		out = append(out, each)
	}
	return out
}
