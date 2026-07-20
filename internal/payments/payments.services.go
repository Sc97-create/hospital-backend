package payments

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/providers"
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
	FulfillmentSvc     *FulfillmentService
}

func NewPaymentsService(
	db *gorm.DB,
	paymentsRepository IPaymentsRepository,
	paymentfactory *providers.PaymentFactory,
	paymentAttempt *SPaymentAttempts,
	prescriptionStatus PrescriptionStatusUpdater,
	fulfillmentSvc *FulfillmentService,
) *PaymentsService {
	return &PaymentsService{
		db:                 db,
		PaymentsRepository: paymentsRepository,
		PaymentFactory:     paymentfactory,
		PaymentAttempt:     paymentAttempt,
		PrescriptionStatus: prescriptionStatus,
		FulfillmentSvc:     fulfillmentSvc,
	}
}

func (p *PaymentsService) CreateLinkPayment(paymentReq dto.CreatePaymentCommand) (paymentRespone dto.CreatePaymentResponse, err error) {

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
	err = p.PrescriptionStatus.UpdateExtPrescriptionStatus(tx, paymentReq.PrescriptionID, constants.StatusPaymentLinkCreated)
	if err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}

// CreatePendingPayment creates a payments row for cash/qr. No gateway call, no payment_attempt.
func (p *PaymentsService) CreatePendingPayment(paymentReq dto.CreatePaymentCommand) error {
	tx := p.db.Begin()
	paymentModel := p.toPaymentModel(paymentReq)
	err := p.PaymentsRepository.Create(tx, paymentModel)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// ConfirmManualPayment confirms cash/qr payment and runs shared fulfillment (no payment_attempt).
func (p *PaymentsService) ConfirmManualPayment(invoiceID, paymentMode, txnRef string) error {
	if invoiceID == "" {
		return fmt.Errorf("invoice_id is required")
	}
	if paymentMode != constants.PaymentCash && paymentMode != constants.PaymentQR {
		return fmt.Errorf("payment_mode must be %s or %s", constants.PaymentCash, constants.PaymentQR)
	}
	_ = txnRef // optional audit field; store later if payments gains a reference column

	payment, err := p.GetPaymentByInvoiceID(invoiceID)
	if err != nil {
		return err
	}
	if payment.Source != constants.PaymentCash && payment.Source != constants.PaymentQR {
		return fmt.Errorf("invoice payment source %s cannot be confirmed manually", payment.Source)
	}
	if paymentMode != payment.Source {
		return fmt.Errorf("payment_mode %s does not match payment source %s", paymentMode, payment.Source)
	}

	tx := p.db.Begin()
	err = p.FulfillmentSvc.FulfillPaidInvoice(tx, invoiceID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if err = tx.Commit().Error; err != nil {
		return err
	}

	paidAt := time.Now()
	return p.FulfillmentSvc.NotifyPaymentReceived(payment, paidAt)
}

func (p *PaymentsService) toPaymentModel(paymentReq dto.CreatePaymentCommand) Payments {
	var payment Payments
	payment.ID = uuid.New().String()
	payment.Amount = paymentReq.Amount
	payment.Channel = paymentReq.Channel
	payment.Source = paymentReq.Source
	payment.CreatedAt = time.Now()
	payment.Currency = paymentReq.Currency
	payment.InvoiceID = paymentReq.InvoiceID
	payment.PatientID = paymentReq.PatientID
	payment.IdempotencyKey = uuid.NewString()
	payment.InitiatedBy = paymentReq.InitiatedBy
	return payment
}

func (p *PaymentsService) FindInvoiceByPaymentAttempt(query string, args ...interface{}) (Payments, error) {
	return p.PaymentsRepository.FindInvoiceByPaymentAttempt(query, args...)
}

func (p *PaymentsService) GetPaymentByInvoiceID(invoiceID string) (Payments, error) {
	if invoiceID == "" {
		return Payments{}, fmt.Errorf("invoice_id is required")
	}
	payment, err := p.PaymentsRepository.FindByInvoiceID(invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Payments{}, fmt.Errorf("payment not found for invoice_id: %s", invoiceID)
		}
		return Payments{}, err
	}
	return payment, nil
}
