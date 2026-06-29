package module

import (
	"hospital-backend/config"

	"hospital-backend/internal/payments"
	"hospital-backend/internal/payments/providers"
	"hospital-backend/internal/payments/providers/razorpay"

	"gorm.io/gorm"
)

type Module struct {
	Paymentservice *payments.PaymentsService
	PaymentAttempt *payments.SPaymentAttempts
}

func NewModule(db *gorm.DB, cfg config.Config) *Module {
	paymentsDB := payments.NewPaymentsDB(db)

	razorpayClient := razorpay.NewClient(
		cfg.RazorPayClient.RPayConfig.BaseUrl,
		cfg.RazorPayClient.RPayConfig.ApiKey,
		cfg.RazorPayClient.RPayConfig.ApiSecret)

	razorpaygateway := razorpay.NewGateway(razorpayClient, cfg.RazorPayClient.CallbackUrl, cfg.RazorPayClient.WebhookSecret)

	gateway := providers.NewPaymentFactory(razorpaygateway)
	paymentAttempts := payments.NewPaymentAttempts(paymentsDB)

	paymentsService := payments.NewPaymentsService(db, paymentsDB, gateway, paymentAttempts)

	return &Module{Paymentservice: paymentsService, PaymentAttempt: paymentAttempts}
}
