package payments

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/providers"
	"hospital-backend/pkg/constants"
	"strings"
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
	if err = validateIdempotencyKey(paymentReq.IdempotencyKey); err != nil {
		return
	}

	// Same client key → return existing payment link (double-click / retry safe)
	if existing, findErr := p.PaymentsRepository.FindByIdempotencyKey(paymentReq.IdempotencyKey); findErr == nil {
		return p.responseFromExistingPayment(existing)
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return paymentRespone, findErr
	}

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
		if isUniqueViolation(err) {
			existing, findErr := p.PaymentsRepository.FindByIdempotencyKey(paymentReq.IdempotencyKey)
			if findErr == nil {
				return p.responseFromExistingPayment(existing)
			}
		}
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
func (p *PaymentsService) CreatePendingPayment(paymentReq dto.CreatePaymentCommand) (Payments, error) {
	if err := validateIdempotencyKey(paymentReq.IdempotencyKey); err != nil {
		return Payments{}, err
	}

	if existing, err := p.PaymentsRepository.FindByIdempotencyKey(paymentReq.IdempotencyKey); err == nil {
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Payments{}, err
	}

	tx := p.db.Begin()
	paymentModel := p.toPaymentModel(paymentReq)
	err := p.PaymentsRepository.Create(tx, paymentModel)
	if err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			existing, findErr := p.PaymentsRepository.FindByIdempotencyKey(paymentReq.IdempotencyKey)
			if findErr == nil {
				return existing, nil
			}
		}
		return Payments{}, err
	}
	if err = tx.Commit().Error; err != nil {
		return Payments{}, err
	}
	return paymentModel, nil
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

func (p *PaymentsService) GetPaymentByIdempotencyKey(key string) (Payments, error) {
	if err := validateIdempotencyKey(key); err != nil {
		return Payments{}, err
	}
	return p.PaymentsRepository.FindByIdempotencyKey(key)
}

func (p *PaymentsService) GetPaymentURLByPaymentID(paymentID string) (string, error) {
	attempt, err := p.PaymentAttempt.FindByPaymentID(paymentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return attempt.PaymentLink, nil
}

func (p *PaymentsService) responseFromExistingPayment(payment Payments) (dto.CreatePaymentResponse, error) {
	attempt, err := p.PaymentAttempt.FindByPaymentID(payment.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.CreatePaymentResponse{}, nil
		}
		return dto.CreatePaymentResponse{}, err
	}
	return dto.CreatePaymentResponse{
		PaymentLinkID: attempt.ProviderLinkID,
		PaymentURL:    attempt.PaymentLink,
		ReferenceID:   attempt.ProviderReferenceID,
	}, nil
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
	payment.IdempotencyKey = paymentReq.IdempotencyKey
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

func validateIdempotencyKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}
