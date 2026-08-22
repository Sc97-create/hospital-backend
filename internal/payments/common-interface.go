package payments

import (
	"context"
	"time"

	invoiceDto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/payments/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/pkg/types"

	"go.uber.org/zap"
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
	UpdateIPrescriptionStatus(tx *gorm.DB, prescriptionItemID string, status string, outOfStock bool) error // item-level status + oos flag
	UpdateExtPrescriptionStatus(tx *gorm.DB, prescriptionID string, status string) error   // prescription-level status
	ResolveAndUpdateParentPrescriptionStatus(tx *gorm.DB, prescriptionID string, medicineInventoryDet []invoiceDto.MedInvoiceItemResponse) error
	UpdateInvoiceStatus(tx *gorm.DB, invoiceID string, status string) error
	GetNotificationPatientByID(patientID string) (map[string]interface{}, error)
	CreateNotification(ctx context.Context, notification notificationdto.CreateRequest) error
}

// PaymentAttemptServicer covers payment attempt persistence used by PaymentsService and webhooks.
type PaymentAttemptServicer interface {
	CreateAttempt(log *zap.Logger, tx *gorm.DB, internalPaymentID string, paymentResponse dto.CreatePaymentResponse, providerName string) error
	CreateAttemptWithIdempotency(log *zap.Logger, tx *gorm.DB, internalPaymentID string, paymentResponse dto.CreatePaymentResponse, providerName string, clientIdempotencyKey string) error
	FindByProviderLinkID(log *zap.Logger, providerLinkID string) (PaymentAttempts, error)
	FindByPaymentID(log *zap.Logger, paymentID string) (PaymentAttempts, error)
	FindByClientIdempotencyKey(log *zap.Logger, key string) (PaymentAttempts, error)
	UpdatePaymentAttempt(log *zap.Logger, tx *gorm.DB, paymentAttempt PaymentAttempts) error
	UpdatePaymentAttemptStatus(log *zap.Logger, tx *gorm.DB, paymentAttempt PaymentAttempts) error
	ClaimForProcessing(log *zap.Logger, providerLinkID string) (PaymentAttempts, bool, error)
}

// FulfillmentServicer orchestrates post-payment side effects.
type FulfillmentServicer interface {
	FulfillPaidInvoice(log *zap.Logger, tx *gorm.DB, invoiceID string) error
	NotifyPaymentReceived(log *zap.Logger, payment Payments, paidAt time.Time) error
}

// PaymentInvoiceLookup resolves payments joined to invoices (webhook path).
type PaymentInvoiceLookup interface {
	FindInvoiceByPaymentAttempt(log *zap.Logger, query string, args ...interface{}) (Payments, error)
}
