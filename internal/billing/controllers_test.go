package billing_test

import (
	"net/http"
	"testing"

	"hospital-backend/internal/billing"
	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/billing/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapErrors "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

const validCheckoutBody = `{
		"patient_id":"pat-1",
		"cashier_id":"cash-1",
		"payment_mode":"cash",
		"organisation_id":"org-1",
		"prescription_id":"rx-1",
		"financials":{"sub_total_amount":100,"tax_amount":10,"total_amount":110},
		"dispense_items":[{
			"medicine_id":"med-1",
			"medicine_inventory_id":"inv-1",
			"prescription_item_id":"pi-1",
			"batch_no":"B1",
			"total_amount":110
		}]
	}`

func TestCheckout(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		headers    map[string]string
		setup      func(*mocks.MockInvoiceServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			headers:    map[string]string{"Idempotency-Key": "key-1"},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing idempotency key",
			body:       validCheckoutBody,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing patient_id",
			body:       `{"cashier_id":"c1","payment_mode":"cash","organisation_id":"o1","financials":{"sub_total_amount":1,"tax_amount":0,"total_amount":1},"prescription_id":"rx-1","dispense_items":[]}`,
			headers:    map[string]string{"Idempotency-Key": "key-1"},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:    "patient not found",
			body:    validCheckoutBody,
			headers: map[string]string{"Idempotency-Key": "key-1"},
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().CreateInvoice(gomock.Any(), gomock.Any()).Return(dto.InvoiceResponse{}, wrapErrors.ErrPatientNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:    "success",
			body:    validCheckoutBody,
			headers: map[string]string{"Idempotency-Key": "key-1"},
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().CreateInvoice(gomock.Any(), gomock.Any()).Return(dto.InvoiceResponse{InvoiceID: "inv-1"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockInvoiceServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			billingCtrl := billing.NewBillingController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/checkout", billingCtrl.Checkout)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method:  http.MethodPost,
				Path:    "/checkout",
				Body:    tt.body,
				Headers: tt.headers,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetInvoiceByPrescriptionID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockInvoiceServicer)
		wantStatus int
	}{
		{
			name: "not found",
			path: "/invoices/prescription/rx-1",
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().GetInvoiceByPrescriptionID(gomock.Any(), gomock.Any()).Return(dto.InvoiceByPrescriptionResponse{}, wrapErrors.ErrInvoiceNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			path: "/invoices/prescription/rx-1",
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().GetInvoiceByPrescriptionID(gomock.Any(), gomock.Any()).Return(dto.InvoiceByPrescriptionResponse{ID: "inv-1"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockInvoiceServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			billingCtrl := billing.NewBillingController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/invoices/prescription/:prescriptionID", billingCtrl.GetInvoiceByPrescriptionID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetInvoiceByAppointmentID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockInvoiceServicer)
		wantStatus int
	}{
		{
			name: "success",
			path: "/invoices/appointment/appt-1",
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().GetInvoiceByAppointmentID(gomock.Any(), gomock.Any()).Return(dto.InvoiceByPrescriptionResponse{ID: "inv-1"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockInvoiceServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			billingCtrl := billing.NewBillingController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/invoices/appointment/:appointmentID", billingCtrl.GetInvoiceByAppointmentID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetBillDetailsByPrescriptionID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockInvoiceServicer)
		wantStatus int
	}{
		{
			name: "prescription not found",
			path: "/bills/prescription/rx-1",
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().GetBillDetailsByPrescriptionID(gomock.Any(), gomock.Any()).Return(dto.BillDetailsByPrescriptionResponse{}, wrapErrors.ErrPrescriptionNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			path: "/bills/prescription/rx-1",
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().GetBillDetailsByPrescriptionID(gomock.Any(), gomock.Any()).Return(dto.BillDetailsByPrescriptionResponse{InvoiceID: "inv-1"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockInvoiceServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			billingCtrl := billing.NewBillingController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/bills/prescription/:prescriptionID", billingCtrl.GetBillDetailsByPrescriptionID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestRetryPaymentLink(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		setup      func(*mocks.MockInvoiceServicer)
		wantStatus int
	}{
		{
			name:       "missing idempotency key",
			path:       "/invoices/inv-1/retry",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:    "invoice not found",
			path:    "/invoices/inv-1/retry",
			headers: map[string]string{"Idempotency-Key": "key-1"},
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().RetryPaymentLink(gomock.Any(), gomock.Any(), gomock.Any()).Return(dto.InvoiceResponse{}, wrapErrors.ErrInvoiceNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:    "success",
			path:    "/invoices/inv-1/retry",
			headers: map[string]string{"Idempotency-Key": "key-1"},
			setup: func(m *mocks.MockInvoiceServicer) {
				m.EXPECT().RetryPaymentLink(gomock.Any(), gomock.Any(), gomock.Any()).Return(dto.InvoiceResponse{InvoiceID: "inv-1", PaymentURL: "http://pay"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockInvoiceServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			billingCtrl := billing.NewBillingController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/invoices/:invoiceID/retry", billingCtrl.RetryPaymentLink)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method:  http.MethodPost,
				Path:    tt.path,
				Headers: tt.headers,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
