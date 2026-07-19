package payments

import (
	"context"
	invoiceDto "hospital-backend/internal/billing/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/pkg/types"

	"gorm.io/gorm"
)

// PrescriptionStatusUpdater is the narrow dependency for PaymentsService.
type PrescriptionStatusUpdater interface {
	UpdateExtPrescriptionStatus(tx *gorm.DB, prescriptionID string, status string) error
}

// IPaymentFulfillment covers post-payment side effects used by the webhook handler.
type IPaymentFulfillment interface {
	GetMedicineInventoryDetByInvoiceID(invoiceID string) ([]invoiceDto.MedInvoiceItemResponse, error)
	// UpdateMedInventoryStock atomically decrements stock — pass dispensedQty, not absolute value
	UpdateMedInventoryStock(tx *gorm.DB, medicineInventoryID string, dispensedQty int64) error
	CreateMedicineMvmt(tx *gorm.DB, medicineMvmt []types.MedicineStockMovements) error
	UpdateDispenseItemQty(tx *gorm.DB, prescriptionItemID string, dispensedQty int64) error
	UpdateIPrescriptionStatus(tx *gorm.DB, prescriptionItemID string, status string) error // item-level status
	UpdateExtPrescriptionStatus(tx *gorm.DB, prescriptionID string, status string) error   // prescription-level status
	UpdateInvoiceStatus(tx *gorm.DB, invoiceID string, status string) error
	GetNotificationPatientByID(patientID string) (map[string]interface{}, error)
	CreateNotification(ctx context.Context, notification notificationdto.CreateRequest) error
}
