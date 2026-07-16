package appinit

import (
	"hospital-backend/internal/billing"
	invoiceDto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/medicine"
	"hospital-backend/internal/prescription"
	"hospital-backend/pkg/types"

	"gorm.io/gorm"
)

// paymentFulfillment adapts billing/medicine/prescription services to payments.IPaymentFulfillment.
type paymentFulfillment struct {
	invoiceItems      *billing.InvoiceItemServ
	invoiceRepo       billing.InvoiceRepo
	medInventory      *medicine.SMedicineInventory
	medMvmt           *medicine.SMedicineMvmt
	prescription      *prescription.PrescriptionService
	prescriptionItems *prescription.PrescriptionItemServ
}

func newPaymentFulfillment(
	invoiceItems *billing.InvoiceItemServ,
	invoiceRepo billing.InvoiceRepo,
	medInventory *medicine.SMedicineInventory,
	medMvmt *medicine.SMedicineMvmt,
	prescriptionSvc *prescription.PrescriptionService,
	prescriptionItems *prescription.PrescriptionItemServ,
) *paymentFulfillment {
	return &paymentFulfillment{
		invoiceItems:      invoiceItems,
		invoiceRepo:       invoiceRepo,
		medInventory:      medInventory,
		medMvmt:           medMvmt,
		prescription:      prescriptionSvc,
		prescriptionItems: prescriptionItems,
	}
}

func (f *paymentFulfillment) GetMedicineInventoryDetByInvoiceID(invoiceID string) ([]invoiceDto.MedInvoiceItemResponse, error) {
	return f.invoiceItems.GetMedicineInventoryDetByInvoiceID(invoiceID)
}

func (f *paymentFulfillment) UpdateMedInventoryStock(tx *gorm.DB, medicineInventoryID string, dispensedQty int64) error {
	return f.medInventory.MedInventory.UpdateMedInventoryStock(tx, medicineInventoryID, dispensedQty)
}

func (f *paymentFulfillment) CreateMedicineMvmt(tx *gorm.DB, medicineMvmt []types.MedicineStockMovements) error {
	return f.medMvmt.CreateMedicineMvmt(tx, medicineMvmt)
}

func (f *paymentFulfillment) UpdateDispenseItemQty(tx *gorm.DB, prescriptionItemID string, dispensedQty int64) error {
	return f.prescriptionItems.UpdateDispenseItemQty(tx, prescriptionItemID, dispensedQty)
}

func (f *paymentFulfillment) UpdateIPrescriptionStatus(tx *gorm.DB, prescriptionItemID string, status string) error {
	return f.prescriptionItems.UpdateIPrescriptionStatus(tx, prescriptionItemID, status)
}

func (f *paymentFulfillment) UpdateExtPrescriptionStatus(tx *gorm.DB, prescriptionID string, status string) error {
	return f.prescription.UpdateExtPrescriptionStatus(tx, prescriptionID, status)
}

func (f *paymentFulfillment) UpdateInvoiceStatus(tx *gorm.DB, invoiceID string, status string) error {
	return f.invoiceRepo.UpdateInvoiceStatus(tx, invoiceID, status)
}
