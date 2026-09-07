package billing

import (
	"errors"
	"testing"

	"hospital-backend/internal/billing/dto"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

func TestValidatePaymentTypeInputs(t *testing.T) {
	svc := &InvoiceServ{}
	log := zap.NewNop()

	if err := svc.validatePaymentTypeInputs(log, PaymentTypePrescription, dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected missing prescription_id error, got %v", err)
	}

	if err := svc.validatePaymentTypeInputs(log, PaymentType("bad"), dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrInvalidPaymentType) {
		t.Fatalf("expected invalid payment type, got %v", err)
	}
}

func TestMapInvoiceCreateErr(t *testing.T) {
	svc := &InvoiceServ{}
	log := zap.NewNop()

	if err := svc.mapInvoiceCreateErr(log, errors.New("db"), PaymentTypePrescription, dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrInvoiceCreateFailed) {
		t.Fatalf("expected create failed, got %v", err)
	}
	if err := svc.mapInvoiceCreateErr(log, errors.New("duplicate key"), PaymentTypePrescription, dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrInvoiceAlreadyExists) {
		t.Fatalf("expected already exists, got %v", err)
	}
	if err := svc.mapInvoiceCreateErr(log, errors.New("unique violation"), PaymentTypeConsultation, dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrAppointmentAlreadyBilled) {
		t.Fatalf("expected appointment already billed, got %v", err)
	}
}

func TestToInvoiceModel(t *testing.T) {
	svc := &InvoiceServ{}
	req := dto.CheckoutReq{
		PatientID:      "pat-1",
		CashierID:      "cash-1",
		OrganisationID: "org-1",
		PrescriptionID: "rx-1",
		Financials: dto.Financial{
			SubtotalAmount: 100,
			TaxAmount:      10,
			TotalAmount:    110,
		},
	}

	inv := svc.toInvoiceModel(req, PaymentTypePrescription)
	if inv.ID == "" || inv.InvoiceCode == "" {
		t.Fatal("expected generated invoice id and code")
	}
	if inv.PaymentType != PaymentTypePrescription || inv.Status != StatusUnpaid {
		t.Fatalf("unexpected invoice fields: %+v", inv)
	}
	if inv.PrescriptionID == nil || *inv.PrescriptionID != "rx-1" {
		t.Fatal("expected prescription id pointer")
	}

	req.AppointmentID = "appt-1"
	inv = svc.toInvoiceModel(req, PaymentTypeConsultation)
	if inv.AppointmentID == nil || *inv.AppointmentID != "appt-1" {
		t.Fatal("expected appointment id pointer for consultation")
	}
}

func TestCreateCode(t *testing.T) {
	svc := &InvoiceServ{}
	if svc.createCode() == "" {
		t.Fatal("expected invoice code")
	}
}

func TestResolvePaymentType(t *testing.T) {
	if resolvePaymentType("") != PaymentTypePrescription {
		t.Fatal("empty should default to prescription")
	}
	if resolvePaymentType("consultation") != PaymentTypeConsultation {
		t.Fatal("expected consultation type")
	}
}

func TestBuildTodayInvoiceCollectionSummary(t *testing.T) {
	summary := buildTodayInvoiceCollectionSummary([]TodayInvoiceCollectionRow{
		{PaymentMode: "cash", Count: 2, Amount: 500},
		{PaymentMode: "qr", Count: 1, Amount: 200},
		{PaymentMode: "link", Count: 3, Amount: 1500},
	})
	if summary.TotalInvoices != 6 || summary.TotalAmount != 2200 {
		t.Fatalf("unexpected totals: %+v", summary)
	}
	if summary.Cash.Count != 2 || summary.Cash.Amount != 500 {
		t.Fatalf("unexpected cash: %+v", summary.Cash)
	}
	if summary.QR.Count != 1 || summary.QR.Amount != 200 {
		t.Fatalf("unexpected qr: %+v", summary.QR)
	}
	if summary.Link.Count != 3 || summary.Link.Amount != 1500 {
		t.Fatalf("unexpected link: %+v", summary.Link)
	}
}
