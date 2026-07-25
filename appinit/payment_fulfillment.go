package appinit

import (
	"context"
	"hospital-backend/internal/billing"
	invoiceDto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/medicine"
	notificationdto "hospital-backend/internal/notifications/dto"
	notificationService "hospital-backend/internal/notifications/service"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/prescription"
	"hospital-backend/pkg/logger"
	"hospital-backend/pkg/types"

	"gorm.io/gorm"
)

// paymentFulfillment adapts billing/medicine/prescription services to payments.IPaymentFulfillment.
type paymentFulfillment struct {
	invoiceItems        *billing.InvoiceItemServ
	invoiceRepo         billing.InvoiceRepo
	medInventory        *medicine.SMedicineInventory
	medMvmt             *medicine.SMedicineMvmt
	prescription        *prescription.PrescriptionService
	prescriptionItems   *prescription.PrescriptionItemServ
	patientService      *patient.PatientService
	notificationService *notificationService.Notificationservice
}

func newPaymentFulfillment(
	invoiceItems *billing.InvoiceItemServ,
	invoiceRepo billing.InvoiceRepo,
	medInventory *medicine.SMedicineInventory,
	medMvmt *medicine.SMedicineMvmt,
	prescriptionSvc *prescription.PrescriptionService,
	prescriptionItems *prescription.PrescriptionItemServ,
	patientService *patient.PatientService,
	notificationService *notificationService.Notificationservice,
) *paymentFulfillment {
	return &paymentFulfillment{
		invoiceItems:        invoiceItems,
		invoiceRepo:         invoiceRepo,
		medInventory:        medInventory,
		medMvmt:             medMvmt,
		prescription:        prescriptionSvc,
		prescriptionItems:   prescriptionItems,
		patientService:      patientService,
		notificationService: notificationService,
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

func (f *paymentFulfillment) ResolveAndUpdateParentPrescriptionStatus(tx *gorm.DB, prescriptionID string, medicineInventoryDet []invoiceDto.MedInvoiceItemResponse) error {
	return f.prescription.ResolveAndUpdateParentStatus(tx, prescriptionID, medicineInventoryDet)
}

func (f *paymentFulfillment) UpdateInvoiceStatus(tx *gorm.DB, invoiceID string, status string) error {
	return f.invoiceRepo.UpdateInvoiceStatus(tx, invoiceID, status)
}
func (f *paymentFulfillment) GetNotificationPatientByID(patientID string) (map[string]interface{}, error) {
	return f.patientService.GetNotificationPatientByID(logger.Log, patientID)
}

func (f *paymentFulfillment) CreateNotification(ctx context.Context, notification notificationdto.CreateRequest) error {
	return f.notificationService.Create(ctx, notification)
}
