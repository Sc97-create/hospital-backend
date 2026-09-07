package billing

import (
	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/prescription"
	wrapError "hospital-backend/shared/error"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InvoiceItemServ struct {
	InvItemRepo      InvoiceItemRepo
	PrescriptionItem prescription.PrescriptionItemServicer
}

func NewInvoiceItemServ(InvoiceItemRepo InvoiceItemRepo, PrescriptionItem prescription.PrescriptionItemServicer) *InvoiceItemServ {
	return &InvoiceItemServ{InvItemRepo: InvoiceItemRepo, PrescriptionItem: PrescriptionItem}
}

func (IItemServ *InvoiceItemServ) addInvoiceItems(log *zap.Logger, db *gorm.DB, prescriptionID string, invoiceID string, invoiceItems []dto.DispensedItem) error {
	log = ensureLog(log)

	prescriptionQtyMap, err := IItemServ.PrescriptionItem.GetqtyByMedicine(log, prescriptionID)
	if err != nil {
		return err
	}

	for _, each := range invoiceItems {
		info, ok := prescriptionQtyMap[each.MedicineID]
		if !ok {
			log.Warn("invoice checkout failed",
				zap.String("prescription_id", prescriptionID),
				zap.String("medicine_id", each.MedicineID),
				zap.String("reason", "medicine_not_in_prescription"),
			)
			return wrapError.ErrMedicineNotInPrescription
		}
		remaining := int64(info.BalanceAfterDispense)
		if int64(each.QuantitySoldUnits) > remaining {
			log.Warn("invoice checkout failed",
				zap.String("prescription_id", prescriptionID),
				zap.String("medicine_id", each.MedicineID),
				zap.String("reason", "qty_exceeds_remaining"),
			)
			return wrapError.ErrQtyExceedsRemaining
		}
		if each.QuantitySoldUnits == 0 {
			continue
		}
		if each.QuantitySoldUnits > each.CurrentStockUnits {
			log.Warn("invoice checkout failed",
				zap.String("prescription_id", prescriptionID),
				zap.String("medicine_id", each.MedicineID),
				zap.String("reason", "insufficient_stock"),
			)
			return wrapError.ErrInsufficientStock
		}
	}

	inoviceItems := IItemServ.toInvoiceItem(invoiceID, invoiceItems)
	err = IItemServ.InvItemRepo.Create(log, db, inoviceItems)
	if err != nil {
		return err
	}
	return nil
}

func (IItemServ *InvoiceItemServ) toInvoiceItem(invoiceID string, items []dto.DispensedItem) []InvoiceItem {
	var InvoiceItems []InvoiceItem
	for _, each := range items {
		var item InvoiceItem
		item.ID = uuid.New().String()
		item.InvoiceID = invoiceID
		item.BatchNo = each.BatchNo
		item.CreatedAt = time.Now()
		item.DispensedQty = int(each.QuantitySoldUnits)
		item.MedicineID = each.MedicineID
		item.MedicineInventoryID = each.MedicineInventoryID
		item.PrescriptionItemID = each.PrescriptionItemID
		item.SubtotalPrice = each.ComputedItemTotal
		item.TotalPrice = each.TotalAmount
		InvoiceItems = append(InvoiceItems, item)
	}
	return InvoiceItems
}

func (IItemServ *InvoiceItemServ) GetMedicineInventoryDetByInvoiceID(log *zap.Logger, invoiceID string) ([]dto.MedInvoiceItemResponse, error) {
	log = ensureLog(log)
	query := `SELECT
    ii.prescription_item_id,
    ii.medicine_inventory_id,
    ii.medicine_id,
    inv.cashier_id,
    inv.organisation_id,
    inv.prescription_id,
    mi.current_stock_units                      AS current_stock_unit,
    ii.dispensed_qty,
    (mi.current_stock_units - ii.dispensed_qty) AS current_stock_unit_after_dispense,
    mi.pricing,
    pi.quantity                                 AS prescribed_qty,
    pi.balance_after_dispense                   AS balance_after_dispense,
	pi.status                                   AS prescription_item_status
	FROM invoice_items ii
	JOIN medicine_inventories mi  ON mi.id  = ii.medicine_inventory_id
	JOIN invoices inv              ON inv.id = ii.invoice_id
	JOIN prescription_items pi    ON pi.id  = ii.prescription_item_id
	WHERE ii.invoice_id = ?`
	invoiceItems, err := IItemServ.InvItemRepo.GetInvoiceItemsByInvoiceID(log, query, invoiceID)
	if err != nil {
		log.Error("invoice items for fulfillment failed",
			zap.String("invoice_id", invoiceID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, wrapError.ErrInvoiceFetchFailed
	}
	log.Debug("invoice items for fulfillment success",
		zap.String("invoice_id", invoiceID),
		zap.Int("item_count", len(invoiceItems)),
	)
	return invoiceItems, nil
}
