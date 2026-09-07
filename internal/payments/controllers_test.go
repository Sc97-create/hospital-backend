package payments_test

import (
	"errors"
	"net/http"
	"testing"

	"hospital-backend/internal/payments"
	"hospital-backend/internal/payments/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapErrors "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestRazorPayWebhook(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		headers    map[string]string
		setup      func(*mocks.MockWebhookServicer)
		wantStatus int
	}{
		{
			name: "unauthorized signature",
			body: `{}`,
			headers: map[string]string{
				"X-Razorpay-Signature": "bad",
			},
			setup: func(m *mocks.MockWebhookServicer) {
				m.EXPECT().ProcessWebhook(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, wrapErrors.ErrInvalidRequest)
			},
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name: "processing error",
			body: `{}`,
			headers: map[string]string{
				"X-Razorpay-Signature": "sig",
			},
			setup: func(m *mocks.MockWebhookServicer) {
				m.EXPECT().ProcessWebhook(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, errors.New("db error"))
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			body: `{}`,
			headers: map[string]string{
				"X-Razorpay-Signature": "sig",
			},
			setup: func(m *mocks.MockWebhookServicer) {
				m.EXPECT().ProcessWebhook(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			webhook := mocks.NewMockWebhookServicer(ctrl)
			if tt.setup != nil {
				tt.setup(webhook)
			}
			paymentCtrl := payments.NewPaymentController(mocks.NewMockPaymentServicer(ctrl), webhook)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/webhooks/razorpay", paymentCtrl.RazorPayWebhook)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method:  http.MethodPost,
				Path:    "/webhooks/razorpay",
				Body:    tt.body,
				Headers: tt.headers,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestUpdatePaymentManually(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockPaymentServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing invoice_id",
			body:       `{"organisation_id":"org-1","payment_mode":"cash"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "payment not found",
			body: `{"invoice_id":"inv-1","organisation_id":"org-1","payment_mode":"cash"}`,
			setup: func(m *mocks.MockPaymentServicer) {
				m.EXPECT().ConfirmManualPayment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(wrapErrors.ErrPaymentNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			body: `{"invoice_id":"inv-1","organisation_id":"org-1","payment_mode":"cash"}`,
			setup: func(m *mocks.MockPaymentServicer) {
				m.EXPECT().ConfirmManualPayment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			payment := mocks.NewMockPaymentServicer(ctrl)
			if tt.setup != nil {
				tt.setup(payment)
			}
			paymentCtrl := payments.NewPaymentController(payment, mocks.NewMockWebhookServicer(ctrl))
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/payments/confirm", paymentCtrl.UpdatePaymentManually)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/payments/confirm",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
