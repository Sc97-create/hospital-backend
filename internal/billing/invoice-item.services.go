package billing

import (
	"fmt"
	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/prescription"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceItemServ struct {
	InvItemRepo      InvoiceItemRepo
	PrescriptionItem *prescription.PrescriptionItemServ
}

func NewInvoiceItemServ(InvoiceItemRepo InvoiceItemRepo, PrescriptionItem *prescription.PrescriptionItemServ) *InvoiceItemServ {
	return &InvoiceItemServ{InvItemRepo: InvoiceItemRepo, PrescriptionItem: PrescriptionItem}
}

func (IItemServ *InvoiceItemServ) addInvoiceItems(db *gorm.DB, prescriptionID string, invoiceID string, invoiceItems []dto.DispensedItem) error {

	prescriptionQtyMap, err := IItemServ.PrescriptionItem.GetqtyByMedicine(prescriptionID)
	if err != nil {
		return err
	}
	// item 4: validate dispensed qty against inventory stock and remaining prescription allowance
	for _, each := range invoiceItems {
		info, ok := prescriptionQtyMap[each.MedicineID]
		if !ok {
			return fmt.Errorf("medicine %s not found in prescription", each.MedicineID)
		}
		remaining := int64(info.BalanceAfterDispense)
		if int64(each.QuantitySoldUnits) > remaining {
			return fmt.Errorf("dispensed qty %.2f exceeds remaining prescribed qty %d for medicine %s",
				each.QuantitySoldUnits, remaining, each.MedicineID)
		}
		// qty 0 = patient skipped this med — skip inventory/stock checks
		if each.QuantitySoldUnits == 0 {
			continue
		}
		if each.QuantitySoldUnits > each.CurrentStockUnits {
			return fmt.Errorf("insufficient stock in batch %s: requested %d, available %d",
				each.BatchNo, each.QuantitySoldUnits, each.CurrentStockUnits)
		}
	}
	inoviceItems := IItemServ.toInvoiceItem(invoiceID, invoiceItems)
	err = IItemServ.InvItemRepo.Create(db, inoviceItems)
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
func (IItemServ *InvoiceItemServ) GetMedicineInventoryDetByInvoiceID(invoiceID string) ([]dto.MedInvoiceItemResponse, error) {
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
	invoiceItems, err := IItemServ.InvItemRepo.GetInvoiceItemsByInvoiceID(query, invoiceID)
	if err != nil {
		return nil, err
	}
	return invoiceItems, nil
}
