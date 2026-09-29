package billing_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/appointments"
	"hospital-backend/internal/billing"
	"hospital-backend/internal/billing/dto"
	billingmocks "hospital-backend/internal/billing/mocks"
	patientdto "hospital-backend/internal/patient/dto"
	"hospital-backend/internal/payments"
	paymentdto "hospital-backend/internal/payments/dto"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type stubPayment struct {
	idemPayment payments.Payments
	idemErr     error
	url         string
	retryResp   paymentdto.CreatePaymentResponse
	retryErr    error
}

func (s stubPayment) GetPaymentByIdempotencyKey(_ *zap.Logger, _ string) (payments.Payments, error) {
	return s.idemPayment, s.idemErr
}

func (s stubPayment) GetPaymentURLByPaymentID(_ *zap.Logger, _ string) (string, error) {
	return s.url, nil
}

func (s stubPayment) CreateLinkPayment(_ *zap.Logger, _ paymentdto.CreatePaymentCommand) (paymentdto.CreatePaymentResponse, error) {
	return paymentdto.CreatePaymentResponse{}, nil
}

func (s stubPayment) CreatePendingPayment(_ *zap.Logger, _ paymentdto.CreatePaymentCommand) (payments.Payments, error) {
	return payments.Payments{}, nil
}

func (s stubPayment) RetryLinkPayment(_ *zap.Logger, _ paymentdto.CreatePaymentCommand) (paymentdto.CreatePaymentResponse, error) {
	return s.retryResp, s.retryErr
}

type stubPatient struct {
	info patientdto.PatientResponse
	err  error
}

func (s stubPatient) FindOne(_ *zap.Logger, _ string) (patientdto.PatientResponse, error) {
	return s.info, s.err
}

type stubAppointment struct {
	appt appointments.Appointment
	err  error
}

func (s stubAppointment) GetAppntmentByID(_ *zap.Logger, _ string) (appointments.Appointment, error) {
	return s.appt, s.err
}

func newInvoiceServ(repo billing.InvoiceRepo, pay billing.PaymentCheckout, patient billing.PatientLookup, appt billing.AppointmentLookup) *billing.InvoiceServ {
	return billing.NewInvoiceServ(nil, repo, pay, nil, patient, appt)
}

func TestCreateInvoiceValidation(t *testing.T) {
	svc := newInvoiceServ(nil, stubPayment{}, nil, nil)

	_, err := svc.CreateInvoice(servicetest.NopLogger(), dto.CheckoutReq{})
	if !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected invalid request for missing idempotency key, got %v", err)
	}

	req := servicetest.ValidCheckoutReq()
	req.PaymentType = "bad"
	req.IdempotencyKey = "idem-1"
	svc = newInvoiceServ(nil, stubPayment{idemErr: gorm.ErrRecordNotFound}, nil, nil)
	_, err = svc.CreateInvoice(servicetest.NopLogger(), req)
	if !errors.Is(err, wrapError.ErrInvalidPaymentType) {
		t.Fatalf("expected invalid payment type, got %v", err)
	}

	req = servicetest.ValidCheckoutReq()
	req.PrescriptionID = ""
	svc = newInvoiceServ(nil, stubPayment{idemErr: gorm.ErrRecordNotFound}, nil, nil)
	_, err = svc.CreateInvoice(servicetest.NopLogger(), req)
	if !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected missing prescription_id, got %v", err)
	}
}

func TestCreateInvoiceIdempotencyReplay(t *testing.T) {
	svc := newInvoiceServ(nil, stubPayment{
		idemPayment: payments.Payments{ID: "pay-1", InvoiceID: "inv-1"},
		url:         "https://pay.example/link",
	}, nil, nil)

	resp, err := svc.CreateInvoice(servicetest.NopLogger(), servicetest.ValidCheckoutReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.InvoiceID != "inv-1" || resp.PaymentURL != "https://pay.example/link" {
		t.Fatalf("unexpected replay response: %+v", resp)
	}
}

func TestCreateInvoiceConsultationValidation(t *testing.T) {
	req := servicetest.ValidCheckoutReq()
	req.PaymentType = "consultation"
	req.PrescriptionID = ""
	req.AppointmentID = ""

	svc := newInvoiceServ(nil, stubPayment{idemErr: gorm.ErrRecordNotFound}, nil, stubAppointment{})
	_, err := svc.CreateInvoice(servicetest.NopLogger(), req)
	if !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected missing appointment_id, got %v", err)
	}

	req.AppointmentID = "appt-1"
	svc = newInvoiceServ(
		nil,
		stubPayment{idemErr: gorm.ErrRecordNotFound},
		nil,
		stubAppointment{appt: appointments.Appointment{ID: "appt-1", OrganisationID: "other", PatientID: "pat-1"}},
	)
	_, err = svc.CreateInvoice(servicetest.NopLogger(), req)
	if !errors.Is(err, wrapError.ErrAppointmentMismatch) {
		t.Fatalf("expected appointment mismatch, got %v", err)
	}
}

func TestServiceGetInvoiceByPrescriptionID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   func(*billingmocks.MockInvoiceRepo)
		wantErr error
	}{
		{name: "empty id", id: "", wantErr: wrapError.ErrInvalidRequest},
		{
			name: "not found",
			id:   "rx-1",
			setup: func(m *billingmocks.MockInvoiceRepo) {
				m.EXPECT().GetInvoiceByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return(billing.InvoiceWithPayment{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrInvoiceNotFound,
		},
		{
			name: "db error",
			id:   "rx-1",
			setup: func(m *billingmocks.MockInvoiceRepo) {
				m.EXPECT().GetInvoiceByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return(billing.InvoiceWithPayment{}, errors.New("db"))
			},
			wantErr: wrapError.ErrInvoiceFetchFailed,
		},
		{
			name: "success",
			id:   "rx-1",
			setup: func(m *billingmocks.MockInvoiceRepo) {
				m.EXPECT().GetInvoiceByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return(billing.InvoiceWithPayment{
					Invoice: billing.Invoice{ID: "inv-1", Status: billing.StatusUnpaid},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := billingmocks.NewMockInvoiceRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newInvoiceServ(repo, nil, nil, nil)
			got, err := svc.GetInvoiceByPrescriptionID(servicetest.NopLogger(), tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != "inv-1" {
				t.Fatalf("expected inv-1, got %q", got.ID)
			}
		})
	}
}

func TestServiceGetInvoiceByAppointmentID(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := billingmocks.NewMockInvoiceRepo(ctrl)
	repo.EXPECT().GetInvoiceByAppointmentID(gomock.Any(), gomock.Any(), "appt-1").Return(billing.InvoiceWithPayment{
		Invoice: billing.Invoice{ID: "inv-1"},
	}, nil)

	svc := newInvoiceServ(repo, nil, nil, nil)
	got, err := svc.GetInvoiceByAppointmentID(servicetest.NopLogger(), "appt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "inv-1" {
		t.Fatalf("expected inv-1, got %q", got.ID)
	}
}

func TestServiceGetBillDetailsByPrescriptionID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   func(*billingmocks.MockInvoiceRepo)
		wantErr error
	}{
		{name: "empty id", id: "", wantErr: wrapError.ErrInvalidRequest},
		{
			name: "not found",
			id:   "rx-1",
			setup: func(m *billingmocks.MockInvoiceRepo) {
				m.EXPECT().GetBillDetailsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return(billing.BillDetailsRow{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrPrescriptionNotFound,
		},
		{
			name: "success",
			id:   "rx-1",
			setup: func(m *billingmocks.MockInvoiceRepo) {
				m.EXPECT().GetBillDetailsByPrescriptionID(gomock.Any(), gomock.Any(), "rx-1").Return(billing.BillDetailsRow{
					PrescriptionID: "rx-1",
					PatientName:    strPtr("Jane"),
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := billingmocks.NewMockInvoiceRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newInvoiceServ(repo, nil, nil, nil)
			got, err := svc.GetBillDetailsByPrescriptionID(servicetest.NopLogger(), tt.id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.PatientDetail.Name != "Jane" {
				t.Fatalf("expected patient Jane, got %q", got.PatientDetail.Name)
			}
		})
	}
}

func TestServiceGetTodayCompletedInvoiceSummary(t *testing.T) {
	tests := []struct {
		name    string
		orgID   string
		setup   func(*billingmocks.MockInvoiceRepo)
		wantErr error
		check   func(t *testing.T, got dto.TodayInvoiceCollectionSummary)
	}{
		{
			name:    "missing organisation id",
			orgID:   "",
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:  "repo error",
			orgID: "org-1",
			setup: func(m *billingmocks.MockInvoiceRepo) {
				m.EXPECT().GetTodayCompletedInvoiceSummary(gomock.Any(), gomock.Any(), "org-1").Return(nil, errors.New("db"))
			},
			wantErr: wrapError.ErrInvoiceFetchFailed,
		},
		{
			name:  "success",
			orgID: "org-1",
			setup: func(m *billingmocks.MockInvoiceRepo) {
				m.EXPECT().GetTodayCompletedInvoiceSummary(gomock.Any(), gomock.Any(), "org-1").Return([]billing.TodayInvoiceCollectionRow{
					{PaymentMode: "cash", Count: 1, Amount: 100},
					{PaymentMode: "link", Count: 2, Amount: 300},
				}, nil)
			},
			check: func(t *testing.T, got dto.TodayInvoiceCollectionSummary) {
				if got.TotalInvoices != 3 || got.TotalAmount != 400 {
					t.Fatalf("unexpected summary: %+v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := billingmocks.NewMockInvoiceRepo(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newInvoiceServ(repo, nil, nil, nil)
			got, err := svc.GetTodayCompletedInvoiceSummary(servicetest.NopLogger(), tt.orgID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected err %v, got %v", tt.wantErr, err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestServiceRetryPaymentLink(t *testing.T) {
	tests := []struct {
		name      string
		invoiceID string
		setup     func(*billingmocks.MockInvoiceRepo, *stubPayment, *stubPatient)
		wantErr   error
	}{
		{
			name:      "missing invoice id",
			invoiceID: "",
			wantErr:   wrapError.ErrInvalidRequest,
		},
		{
			name:      "invoice not found",
			invoiceID: "inv-1",
			setup: func(repo *billingmocks.MockInvoiceRepo, _ *stubPayment, _ *stubPatient) {
				repo.EXPECT().GetInvoiceByID(gomock.Any(), "inv-1").Return(billing.Invoice{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrInvoiceNotFound,
		},
		{
			name:      "patient not found",
			invoiceID: "inv-1",
			setup: func(repo *billingmocks.MockInvoiceRepo, _ *stubPayment, patient *stubPatient) {
				repo.EXPECT().GetInvoiceByID(gomock.Any(), "inv-1").Return(billing.Invoice{
					ID: "inv-1", PatientID: "pat-1", Status: billing.StatusUnpaid,
				}, nil)
				patient.err = wrapError.ErrPatientNotFound
			},
			wantErr: wrapError.ErrPatientNotFound,
		},
		{
			name:      "success",
			invoiceID: "inv-1",
			setup: func(repo *billingmocks.MockInvoiceRepo, pay *stubPayment, patient *stubPatient) {
				repo.EXPECT().GetInvoiceByID(gomock.Any(), "inv-1").Return(billing.Invoice{
					ID: "inv-1", PatientID: "pat-1", Status: billing.StatusUnpaid, InvoiceCode: "INV-1",
				}, nil)
				patient.info = patientdto.PatientResponse{PatientEmail: "j@example.com", PatientPhone: "999"}
				pay.retryResp = paymentdto.CreatePaymentResponse{PaymentURL: "https://pay/link"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := billingmocks.NewMockInvoiceRepo(ctrl)
			pay := &stubPayment{}
			patient := &stubPatient{}
			if tt.setup != nil {
				tt.setup(repo, pay, patient)
			}
			svc := newInvoiceServ(repo, pay, patient, nil)
			resp, err := svc.RetryPaymentLink(servicetest.NopLogger(), tt.invoiceID, "idem-1")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.PaymentURL == "" {
				t.Fatal("expected payment url")
			}
		})
	}
}

func strPtr(v string) *string {
	return &v
}
