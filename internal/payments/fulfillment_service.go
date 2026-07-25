package payments

import (
	"context"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/pkg/constants"
	"hospital-backend/pkg/types"
	"time"

	"github.com/google/uuid"
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
func (s *FulfillmentService) FulfillPaidInvoice(tx *gorm.DB, invoiceID string) error {
	medicineInventoryDet, err := s.Deps.GetMedicineInventoryDetByInvoiceID(invoiceID)
	if err != nil {
		return err
	}
	var prescriptionID string
	if len(medicineInventoryDet) > 0 {
		prescriptionID = medicineInventoryDet[0].PrescriptionID
	}

	var bulkMedicineMvmt []types.MedicineStockMovements
	for i, each := range medicineInventoryDet {
		// qty 0 = skipped med — no inventory decrement or stock movement
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
				return err
			}
		}

		// remaining = balance_after_dispense - this dispense (e.g. qty 12, dispense 5 → 7)
		err = s.Deps.UpdateDispenseItemQty(tx, each.PrescriptionItemID, each.DispensedQty)
		if err != nil {
			return err
		}

		remaining := each.BalanceAfterDispense - each.DispensedQty
		if remaining != 0 {
			medicineInventoryDet[i].PrescriptionItemStatus = constants.StatusPartiallyDispensed
			err = s.Deps.UpdateIPrescriptionStatus(tx, each.PrescriptionItemID, constants.StatusPartiallyDispensed)
		} else {
			medicineInventoryDet[i].PrescriptionItemStatus = constants.StatusFullyDispensed
			err = s.Deps.UpdateIPrescriptionStatus(tx, each.PrescriptionItemID, constants.StatusFullyDispensed)
		}
		if err != nil {
			return err
		}
	}

	if len(bulkMedicineMvmt) > 0 {
		err = s.Deps.CreateMedicineMvmt(tx, bulkMedicineMvmt)
		if err != nil {
			return err
		}
	}

	if prescriptionID != "" {
		err = s.Deps.ResolveAndUpdateParentPrescriptionStatus(tx, prescriptionID, medicineInventoryDet)
		if err != nil {
			return err
		}
	}

	return s.Deps.UpdateInvoiceStatus(tx, invoiceID, constants.InvoicePaid)
}

// NotifyPaymentReceived sends the same payment_received notification used after webhook paid.
// Call after the fulfillment transaction has committed.
func (s *FulfillmentService) NotifyPaymentReceived(payment Payments, paidAt time.Time) error {
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
