package billing

import (
	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/prescription"
	prescriptionDTO "hospital-backend/internal/prescription/dto"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceItemServ struct {
	InvItemRepo      InvoiceItemRepo
	PrescriptionItem prescription.PrescriptionItemServ
}

func NewInvoiceItemServ(InvoiceItemRepo InvoiceItemRepo) *InvoiceItemServ {
	return &InvoiceItemServ{InvItemRepo: InvoiceItemRepo}
}

func (IItemServ *InvoiceItemServ) addInvoiceItems(db *gorm.DB, prescriptionID string, invoiceID string, invoiceItems []dto.DispensedItem) error {
	prescriptionQtyMap, err := IItemServ.PrescriptionItem.GetqtyByMedicine(prescriptionID)
	if err != nil {
		return err
	}
	inoviceItems := IItemServ.toInvoiceItem(prescriptionQtyMap, invoiceID, invoiceItems)
	err = IItemServ.InvItemRepo.Create(db, inoviceItems)
	if err != nil {
		return err
	}
	return nil
}

func (IItemServ *InvoiceItemServ) toInvoiceItem(prescriptionQtyMap map[string]prescriptionDTO.PrescriptionQtyInfo, invoiceID string, items []dto.DispensedItem) []InvoiceItem {
	var InvoiceItems []InvoiceItem
	for _, each := range items {
		var item InvoiceItem
		item.ID = uuid.New().String()
		item.BatchNo = each.BatchNo
		item.CreatedAt = time.Now()
		item.DispensedQty = int(each.QuantitySoldUnits)
		item.MedicineID = each.MedicineID
		item.SubtotalPrice = each.ComputedItemTotal
		item.TotalPrice = each.TotalAmount
		item.Pendingqty = prescriptionQtyMap[each.MedicineID].Quantity - int(each.QuantitySoldUnits) // need to take from db
		InvoiceItems = append(InvoiceItems, item)
	}
	return InvoiceItems
}
