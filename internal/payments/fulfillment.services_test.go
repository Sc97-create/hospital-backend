package payments_test

import (
	"context"
	"errors"
	"testing"
	"time"

	invoiceDto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/payments"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/constants"
	"hospital-backend/pkg/types"
	wrapError "hospital-backend/shared/error"

	"gorm.io/gorm"
)

type stubFulfillmentDeps struct {
	inventoryItems []invoiceDto.MedInvoiceItemResponse
	inventoryErr   error

	updateInvoiceID     string
	updateInvoiceStatus string
	updateInvoiceErr    error
	updateInvoiceCalled bool

	patientInfo map[string]interface{}
	patientErr  error

	notificationReq     notificationdto.CreateRequest
	notificationErr     error
	notificationCalled  bool
}

func (s *stubFulfillmentDeps) GetMedicineInventoryDetByInvoiceID(_ string) ([]invoiceDto.MedInvoiceItemResponse, error) {
	return s.inventoryItems, s.inventoryErr
}

func (s *stubFulfillmentDeps) UpdateMedInventoryStock(_ *gorm.DB, _ string, _ int64) error {
	return nil
}

func (s *stubFulfillmentDeps) CreateMedicineMvmt(_ *gorm.DB, _ []types.MedicineStockMovements) error {
	return nil
}

func (s *stubFulfillmentDeps) UpdateDispenseItemQty(_ *gorm.DB, _ string, _ int64) error {
	return nil
}

func (s *stubFulfillmentDeps) UpdateIPrescriptionStatus(_ *gorm.DB, _ string, _ string, _ bool) error {
	return nil
}

func (s *stubFulfillmentDeps) UpdateExtPrescriptionStatus(_ *gorm.DB, _ string, _ string) error {
	return nil
}

func (s *stubFulfillmentDeps) ResolveAndUpdateParentPrescriptionStatus(_ *gorm.DB, _ string, _ []invoiceDto.MedInvoiceItemResponse) error {
	return nil
}

func (s *stubFulfillmentDeps) UpdateInvoiceStatus(_ *gorm.DB, invoiceID, status string) error {
	s.updateInvoiceCalled = true
	s.updateInvoiceID = invoiceID
	s.updateInvoiceStatus = status
	return s.updateInvoiceErr
}

func (s *stubFulfillmentDeps) GetNotificationPatientByID(_ string) (map[string]interface{}, error) {
	return s.patientInfo, s.patientErr
}

func (s *stubFulfillmentDeps) CreateNotification(_ context.Context, notification notificationdto.CreateRequest) error {
	s.notificationCalled = true
	s.notificationReq = notification
	return s.notificationErr
}

func TestFulfillPaidInvoice(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("inventory error", func(t *testing.T) {
		deps := &stubFulfillmentDeps{inventoryErr: errors.New("inventory lookup failed")}
		svc := payments.NewFulfillmentService(deps)
		err := svc.FulfillPaidInvoice(log, nil, "inv-1")
		if !errors.Is(err, wrapError.ErrPaymentFulfillFailed) {
			t.Fatalf("expected fulfill failed, got %v", err)
		}
	})

	t.Run("success minimal path empty items updates invoice", func(t *testing.T) {
		deps := &stubFulfillmentDeps{inventoryItems: []invoiceDto.MedInvoiceItemResponse{}}
		svc := payments.NewFulfillmentService(deps)
		if err := svc.FulfillPaidInvoice(log, nil, "inv-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !deps.updateInvoiceCalled {
			t.Fatal("expected UpdateInvoiceStatus to be called")
		}
		if deps.updateInvoiceID != "inv-1" || deps.updateInvoiceStatus != constants.InvoicePaid {
			t.Fatalf("unexpected invoice update: id=%s status=%s", deps.updateInvoiceID, deps.updateInvoiceStatus)
		}
	})
}

func TestNotifyPaymentReceived(t *testing.T) {
	log := servicetest.NopLogger()
	payment := payments.Payments{
		ID:        "pay-1",
		PatientID: "pat-1",
		Amount:    250,
		Currency:  "INR",
	}
	paidAt := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)

	t.Run("patient lookup error", func(t *testing.T) {
		deps := &stubFulfillmentDeps{patientErr: errors.New("patient not found")}
		svc := payments.NewFulfillmentService(deps)
		if err := svc.NotifyPaymentReceived(log, payment, paidAt); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		deps := &stubFulfillmentDeps{
			patientInfo: map[string]interface{}{
				"patient_id": "pat-1",
				"name":       "Jane Doe",
			},
		}
		svc := payments.NewFulfillmentService(deps)
		if err := svc.NotifyPaymentReceived(log, payment, paidAt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !deps.notificationCalled {
			t.Fatal("expected CreateNotification to be called")
		}
		if deps.notificationReq.NotificationType != constants.PaymentReceivedEvent {
			t.Fatalf("unexpected notification type: %s", deps.notificationReq.NotificationType)
		}
		data, ok := deps.notificationReq.Data.(map[string]interface{})
		if !ok || data["amount_paid"] != payment.Amount {
			t.Fatalf("unexpected notification data: %+v", deps.notificationReq.Data)
		}
	})
}
