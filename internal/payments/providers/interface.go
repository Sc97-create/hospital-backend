package providers

import (
	"context"
	"hospital-backend/internal/payments/dto"
)

const ()

type Provider interface {
	Name() string
	CreatePayment(ctx context.Context, req dto.CreatePaymentCommand) (dto.CreatePaymentResponse, error)
	VerifySignature(payload []byte, signature string) (bool, error)
	//GetPayment(ctx context.Context, paymentID string) (dto.GetPaymentResponse, error)
	// UpdatePayment(ctx context.Context, paymentID string, req dto.UpdatePaymentRequest) (dto.UpdatePaymentResponse, error)
	// DeletePayment(ctx context.Context, paymentID string) error
}
