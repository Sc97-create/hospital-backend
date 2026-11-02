package payments

import (
	"errors"
	"testing"

	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecordSubscriptionPayment(t *testing.T) {
	log := servicetest.NopLogger()
	db, err := gorm.Open(sqlite.Open("file:subpay?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Payments{}, &PaymentAttempts{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := &PaymentsService{db: db}
	invoiceID := uuid.NewString()
	customerID := uuid.NewString()

	_, err = svc.RecordSubscriptionPayment(log, dto.SubscriptionPaymentRequest{Amount: 10})
	if !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("invalid err=%v", err)
	}

	got, err := svc.RecordSubscriptionPayment(log, dto.SubscriptionPaymentRequest{
		InvoiceID:  invoiceID,
		CustomerID: customerID,
		Amount:     5400,
		OrderID:    "order_sub",
		Currency:   constants.IndCurrnecy,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	again, err := svc.RecordSubscriptionPayment(log, dto.SubscriptionPaymentRequest{
		InvoiceID:  invoiceID,
		CustomerID: customerID,
		Amount:     5400,
		OrderID:    "order_sub",
		Currency:   constants.IndCurrnecy,
	})
	if err != nil || again.PaymentID != got.PaymentID {
		t.Fatalf("replay got=%+v err=%v", again, err)
	}

	var row Payments
	if err := db.Where("id = ?", got.PaymentID).First(&row).Error; err != nil {
		t.Fatalf("read: %v", err)
	}
	if row.PatientID != "" || row.CustomerID == nil || *row.CustomerID != customerID || row.Source != constants.PaymentLink || row.Channel != constants.PaymentCard || row.OrderID != "order_sub" {
		t.Fatalf("row=%+v", row)
	}
	var attempt PaymentAttempts
	if err := db.Where("payment_id = ?", got.PaymentID).First(&attempt).Error; err != nil {
		t.Fatalf("attempt: %v", err)
	}
	if attempt.ProviderOrderID != "order_sub" || attempt.Provider != subscriptionProvider {
		t.Fatalf("attempt=%+v", attempt)
	}
}
