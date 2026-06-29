package payments

import (
	"hospital-backend/internal/payments/dto"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SPaymentAttempts struct {
	PaymentAttemptRepo IPaymentAttempts
}

func NewPaymentAttempts(paymentARepo IPaymentAttempts) *SPaymentAttempts {
	return &SPaymentAttempts{PaymentAttemptRepo: paymentARepo}
}
func (sPAttempts *SPaymentAttempts) CreateAttempt(tx *gorm.DB, paymentID string, paymentResponse dto.CreatePaymentResponse, providerName string) error {
	paymentAttempts := sPAttempts.toPaymentAModel(paymentID, paymentResponse, providerName)
	err := sPAttempts.PaymentAttemptRepo.CreatePaymentAttempts(tx, paymentAttempts)
	if err != nil {
		return err
	}
	return nil
}
func (sPAttempts *SPaymentAttempts) toPaymentAModel(paymentID string, paymentResponse dto.CreatePaymentResponse, providerName string) PaymentAttempts {
	var PayAttempts PaymentAttempts
	PayAttempts.ID = uuid.NewString()
	PayAttempts.AttemptNo = 1
	PayAttempts.PaymentLink = paymentResponse.PaymentURL
	PayAttempts.PaymentID = paymentResponse.PaymentID
	PayAttempts.Provider = providerName
	PayAttempts.Status = StatusPending
	PayAttempts.CreatedAt = time.Now()
	PayAttempts.PaymentID = paymentID
	PayAttempts.ProviderRequest = paymentResponse.RequestPayload
	return PayAttempts
}
