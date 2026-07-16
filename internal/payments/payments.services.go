package payments

import (
	"context"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/providers"
	"hospital-backend/internal/prescription"
	"hospital-backend/pkg/constants"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentsService struct {
	db                 *gorm.DB
	PaymentsRepository IPaymentsRepository
	PaymentFactory     *providers.PaymentFactory
	PaymentAttempt     *SPaymentAttempts
	PrescriptionStatus PrescriptionStatusUpdater
}

func NewPaymentsService(db *gorm.DB, paymentsRepository IPaymentsRepository, paymentfactory *providers.PaymentFactory, paymentAttempt *SPaymentAttempts, prescriptionStatus PrescriptionStatusUpdater) *PaymentsService {
	return &PaymentsService{db: db, PaymentsRepository: paymentsRepository, PaymentFactory: paymentfactory, PaymentAttempt: paymentAttempt, PrescriptionStatus: prescriptionStatus}
}

func (p *PaymentsService) StorePaymentandNotifyUser(paymentReq dto.CreatePaymentCommand) (paymentRespone dto.CreatePaymentResponse, err error) {

	provider, err := p.PaymentFactory.GetProvider(constants.ProviderNameRazorpay)
	if err != nil {
		return
	}
	providerName := provider.Name()
	ctx, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()

	paymentRespone, err = provider.CreatePayment(ctx, paymentReq)
	if err != nil {
		return
	}
	//take paymentreq from creatpayment response
	tx := p.db.Begin()
	paymentModel := p.toPaymentModel(paymentReq)
	err = p.PaymentsRepository.Create(tx, paymentModel)
	if err != nil {
		tx.Rollback()
		return
	}
	err = p.PaymentAttempt.CreateAttempt(tx, paymentModel.ID, paymentRespone, providerName)
	if err != nil {
		tx.Rollback()
		return
	}
	err = p.PrescriptionStatus.UpdateExtPrescriptionStatus(tx, paymentReq.PrescriptionID, string(prescription.StatusPaymentLinkCreated))
	if err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}
func (p *PaymentsService) toPaymentModel(paymentReq dto.CreatePaymentCommand) Payments {
	var payment Payments
	payment.ID = uuid.New().String()
	payment.Amount = paymentReq.Amount
	payment.Channel = paymentReq.Channel
	payment.Source = paymentReq.Source
	payment.CreatedAt = time.Now()
	payment.Currency = paymentReq.Currency
	//payment.ExpiresAt = paymentReq.ExpiresAt
	payment.InvoiceID = paymentReq.InvoiceID
	payment.PatientID = paymentReq.PatientID
	payment.IdempotencyKey = uuid.NewString()
	payment.InitiatedBy = paymentReq.InitiatedBy
	//payment.Status = StatusPending
	return payment
}

func (p *PaymentsService) FindInvoiceByPaymentAttempt(query string, args ...interface{}) (Payments, error) {
	return p.PaymentsRepository.FindInvoiceByPaymentAttempt(query, args...)
}
