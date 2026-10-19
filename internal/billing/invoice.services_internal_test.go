package billing

import (
	"errors"
	"testing"

	"hospital-backend/internal/billing/dto"
	patientdto "hospital-backend/internal/patient/dto"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

func TestValidatePaymentTypeInputs(t *testing.T) {
	svc := &InvoiceServ{}
	log := zap.NewNop()

	if err := svc.validatePaymentTypeInputs(log, PaymentTypePrescription, dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected missing billing_id error, got %v", err)
	}

	if err := svc.validatePaymentTypeInputs(log, PaymentType("bad"), dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrInvalidPaymentType) {
		t.Fatalf("expected invalid payment type, got %v", err)
	}

	if err := svc.validatePaymentTypeInputs(log, PaymentTypeLabOrder, dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected missing billing_id for lab_order, got %v", err)
	}
}

func TestValidateFinancials(t *testing.T) {
	log := zap.NewNop()

	if err := validateFinancials(log, dto.CheckoutReq{Financials: dto.Financial{TotalAmount: 0}}); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected reject zero total, got %v", err)
	}
	if err := validateFinancials(log, dto.CheckoutReq{Financials: dto.Financial{TotalAmount: -1}}); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected reject negative total, got %v", err)
	}
	if err := validateFinancials(log, dto.CheckoutReq{Financials: dto.Financial{SubtotalAmount: -1, TotalAmount: 10}}); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected reject negative subtotal, got %v", err)
	}
	if err := validateFinancials(log, dto.CheckoutReq{Financials: dto.Financial{TaxAmount: -1, TotalAmount: 10}}); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected reject negative tax, got %v", err)
	}
	if err := validateFinancials(log, dto.CheckoutReq{Financials: dto.Financial{DiscountAmount: -5, TotalAmount: 10}}); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected reject negative discount, got %v", err)
	}
	if err := validateFinancials(log, dto.CheckoutReq{Financials: dto.Financial{SubtotalAmount: 100, TaxAmount: 0, DiscountAmount: 0, TotalAmount: 100}}); err != nil {
		t.Fatalf("expected valid financials, got %v", err)
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
	if err := svc.mapInvoiceCreateErr(log, errors.New("unique violation"), PaymentTypeLabOrder, dto.CheckoutReq{}); !errors.Is(err, wrapError.ErrLabOrderAlreadyBilled) {
		t.Fatalf("expected lab order already billed, got %v", err)
	}
}

func TestToInvoiceModel(t *testing.T) {
	svc := &InvoiceServ{}
	req := dto.CheckoutReq{
		PatientID:      "pat-1",
		CashierID:      "cash-1",
		OrganisationID: "org-1",
		BillingID:      "rx-1",
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
	if inv.BillingID != "rx-1" {
		t.Fatal("expected billing id for prescription")
	}

	req.BillingID = "appt-1"
	inv = svc.toInvoiceModel(req, PaymentTypeConsultation)
	if inv.BillingID != "appt-1" || inv.PaymentType != PaymentTypeConsultation {
		t.Fatal("expected billing id for consultation")
	}
}

func TestToPaymentlinkModelBillingID(t *testing.T) {
	svc := &InvoiceServ{}
	patient := patientdto.PatientResponse{
		PatientID:    "pat-1",
		PatientName:  "Ada",
		PatientEmail: "ada@example.com",
		PatientPhone: "999",
	}
	req := dto.CheckoutReq{
		BillingID:      "rx-billing-1",
		CashierID:      "cash-1",
		PaymentMode:    "upi",
		IdempotencyKey: "idem-rx",
		Financials:     dto.Financial{TotalAmount: 150.4},
	}

	cmd := svc.toPaymentlinkModel(req, patient, "inv-1", "INV-1", PaymentTypePrescription)
	if cmd.InvoiceID != "inv-1" {
		t.Fatalf("expected invoice_id inv-1, got %q", cmd.InvoiceID)
	}
	if cmd.PrescriptionID != "rx-billing-1" {
		t.Fatalf("prescription payment must carry billing_id as prescription_id, got %q", cmd.PrescriptionID)
	}

	req.BillingID = "appt-billing-1"
	req.IdempotencyKey = "idem-appt"
	cmd = svc.toPaymentlinkModel(req, patient, "inv-2", "INV-2", PaymentTypeConsultation)
	if cmd.PrescriptionID != "" {
		t.Fatalf("consultation payment must not set prescription_id from billing_id, got %q", cmd.PrescriptionID)
	}
	if cmd.InvoiceID != "inv-2" {
		t.Fatalf("expected invoice_id inv-2, got %q", cmd.InvoiceID)
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
	if resolvePaymentType("lab_order") != PaymentTypeLabOrder {
		t.Fatal("expected lab_order type")
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
