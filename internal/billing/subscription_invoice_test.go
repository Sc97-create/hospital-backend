package billing

import (
	"errors"
	"testing"

	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateSubscriptionInvoice(t *testing.T) {
	log := servicetest.NopLogger()
	db, err := gorm.Open(sqlite.Open("file:subinvoice?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Invoice{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := &InvoiceServ{db: db}
	subscriptionID := uuid.NewString()
	customerID := uuid.NewString()
	organisationID := uuid.NewString()

	_, err = svc.CreateSubscriptionInvoice(log, dto.SubscriptionInvoiceRequest{Amount: 10})
	if !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("invalid err=%v", err)
	}

	first, err := svc.CreateSubscriptionInvoice(log, dto.SubscriptionInvoiceRequest{
		SubscriptionID: subscriptionID,
		CustomerID:     customerID,
		OrganisationID: organisationID,
		Amount:         5400,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if first.PaymentType != string(PaymentTypeSubscriptionNew) || first.Status != StatusUnpaid || first.InvoiceID == "" {
		t.Fatalf("first=%+v", first)
	}

	var stored Invoice
	if err := db.Where("id = ?", first.InvoiceID).First(&stored).Error; err != nil {
		t.Fatalf("read: %v", err)
	}
	if stored.PatientID != "" || stored.CashierID != "" || stored.CustomerID == nil || *stored.CustomerID != customerID {
		t.Fatalf("party patient=%q cashier=%q customer=%v", stored.PatientID, stored.CashierID, stored.CustomerID)
	}
	if stored.PaymentType != PaymentTypeSubscriptionNew || stored.BillingID != subscriptionID {
		t.Fatalf("invoice=%+v", stored)
	}

	_, err = svc.CreateSubscriptionInvoice(log, dto.SubscriptionInvoiceRequest{
		SubscriptionID: subscriptionID,
		CustomerID:     customerID,
		OrganisationID: organisationID,
		Amount:         5400,
	})
	if !errors.Is(err, wrapError.ErrInvoiceAlreadyExists) {
		t.Fatalf("duplicate new err=%v", err)
	}

	renewal, err := svc.CreateSubscriptionInvoice(log, dto.SubscriptionInvoiceRequest{
		SubscriptionID: subscriptionID,
		CustomerID:     customerID,
		OrganisationID: organisationID,
		Amount:         1000,
		Renewal:        true,
	})
	if err != nil {
		t.Fatalf("renewal: %v", err)
	}
	_, err = svc.CreateSubscriptionInvoice(log, dto.SubscriptionInvoiceRequest{
		SubscriptionID: subscriptionID,
		CustomerID:     customerID,
		OrganisationID: organisationID,
		Amount:         1000,
		Renewal:        true,
	})
	if !errors.Is(err, wrapError.ErrInvoiceAlreadyExists) {
		t.Fatalf("unpaid renewal err=%v", err)
	}
	if err := db.Model(&Invoice{}).Where("id = ?", renewal.InvoiceID).Update("status", StatusPaid).Error; err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	if _, err = svc.CreateSubscriptionInvoice(log, dto.SubscriptionInvoiceRequest{
		SubscriptionID: subscriptionID,
		CustomerID:     customerID,
		OrganisationID: organisationID,
		Amount:         1000,
		Renewal:        true,
	}); err != nil {
		t.Fatalf("second renewal: %v", err)
	}
}

func TestCreateSubscriptionInvoiceAfterTrialIsNew(t *testing.T) {
	log := servicetest.NopLogger()
	db, err := gorm.Open(sqlite.Open("file:subinvoice-trial?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Invoice{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := &InvoiceServ{db: db}

	// A free trial stores no invoice. The first paid change is sent as a renewal.
	got, err := svc.CreateSubscriptionInvoice(log, dto.SubscriptionInvoiceRequest{
		SubscriptionID: uuid.NewString(),
		CustomerID:     uuid.NewString(),
		OrganisationID: uuid.NewString(),
		Amount:         5400,
		Renewal:        true,
	})
	if err != nil {
		t.Fatalf("after trial: %v", err)
	}
	if got.PaymentType != string(PaymentTypeSubscriptionNew) || got.Status != StatusUnpaid {
		t.Fatalf("after trial got=%+v", got)
	}
}
