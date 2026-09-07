package payments

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/internal/payments/dto"
	"hospital-backend/internal/payments/providers"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PaymentsService struct {
	db                 *gorm.DB
	PaymentsRepository IPaymentsRepository
	PaymentFactory     providers.IPaymentFactory
	PaymentAttempt     PaymentAttemptServicer
	PrescriptionStatus PrescriptionStatusUpdater
	FulfillmentSvc     FulfillmentServicer
}

func NewPaymentsService(
	db *gorm.DB,
	paymentsRepository IPaymentsRepository,
	paymentfactory providers.IPaymentFactory,
	paymentAttempt PaymentAttemptServicer,
	prescriptionStatus PrescriptionStatusUpdater,
	fulfillmentSvc FulfillmentServicer,
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

func (p *PaymentsService) CreateLinkPayment(log *zap.Logger, paymentReq dto.CreatePaymentCommand) (paymentRespone dto.CreatePaymentResponse, err error) {
	log = ensureLog(log)
	if err = validateIdempotencyKey(paymentReq.IdempotencyKey); err != nil {
		log.Warn("payment link create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "idempotency_required"),
		)
		return
	}

	if existing, findErr := p.PaymentsRepository.FindByIdempotencyKey(log, paymentReq.IdempotencyKey); findErr == nil {
		resp, respErr := p.responseFromExistingPayment(log, existing)
		if respErr == nil {
			log.Info("payment link create success",
				zap.String("payment_id", existing.ID),
				zap.String("invoice_id", existing.InvoiceID),
				zap.String("idempotency_key", paymentReq.IdempotencyKey),
				zap.Bool("idempotent_replay", true),
				zap.Bool("has_payment_url", resp.PaymentURL != ""),
			)
		}
		return resp, respErr
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		log.Error("payment link create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "idempotency_lookup"),
			zap.Error(findErr),
		)
		return paymentRespone, findErr
	}

	provider, err := p.PaymentFactory.GetProvider(constants.ProviderNameRazorpay)
	if err != nil {
		log.Error("payment link create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "provider"),
			zap.Error(err),
		)
		return
	}
	providerName := provider.Name()
	ctx, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()

	paymentRespone, err = provider.CreatePayment(ctx, paymentReq)
	if err != nil {
		log.Error("payment link create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "gateway_create"),
			zap.Error(err),
		)
		return
	}

	tx := p.db.Begin()
	paymentModel := p.toPaymentModel(paymentReq)
	err = p.PaymentsRepository.Create(log, tx, paymentModel)
	if err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			existing, findErr := p.PaymentsRepository.FindByIdempotencyKey(log, paymentReq.IdempotencyKey)
			if findErr == nil {
				resp, respErr := p.responseFromExistingPayment(log, existing)
				if respErr == nil {
					log.Info("payment link create success",
						zap.String("payment_id", existing.ID),
						zap.String("invoice_id", existing.InvoiceID),
						zap.String("idempotency_key", paymentReq.IdempotencyKey),
						zap.Bool("idempotent_replay", true),
						zap.Bool("has_payment_url", resp.PaymentURL != ""),
					)
				}
				return resp, respErr
			}
		}
		log.Error("payment link create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return
	}
	err = p.PaymentAttempt.CreateAttempt(log, tx, paymentModel.ID, paymentRespone, providerName)
	if err != nil {
		tx.Rollback()
		log.Error("payment link create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("payment_id", paymentModel.ID),
			zap.String("reason", "db_attempt"),
			zap.Error(err),
		)
		return
	}
	err = p.PrescriptionStatus.UpdateExtPrescriptionStatus(tx, paymentReq.PrescriptionID, constants.StatusPaymentPending)
	if err != nil {
		tx.Rollback()
		log.Error("payment link create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("prescription_id", paymentReq.PrescriptionID),
			zap.String("reason", "prescription_status"),
			zap.Error(err),
		)
		return
	}
	tx.Commit()
	log.Info("payment link create success",
		zap.String("payment_id", paymentModel.ID),
		zap.String("invoice_id", paymentReq.InvoiceID),
		zap.String("prescription_id", paymentReq.PrescriptionID),
		zap.String("provider_link_id", paymentRespone.PaymentLinkID),
		zap.String("idempotency_key", paymentReq.IdempotencyKey),
		zap.Bool("has_payment_url", paymentRespone.PaymentURL != ""),
	)
	return
}

func (p *PaymentsService) RetryLinkPayment(log *zap.Logger, paymentReq dto.CreatePaymentCommand) (dto.CreatePaymentResponse, error) {
	log = ensureLog(log)
	if err := validateIdempotencyKey(paymentReq.IdempotencyKey); err != nil {
		log.Warn("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "idempotency_required"),
		)
		return dto.CreatePaymentResponse{}, err
	}
	if paymentReq.InvoiceID == "" || paymentReq.PrescriptionID == "" {
		log.Warn("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "missing_ids"),
		)
		return dto.CreatePaymentResponse{}, fmt.Errorf("invoice_id and prescription_id are required")
	}

	if existingAttempt, findErr := p.PaymentAttempt.FindByClientIdempotencyKey(log, paymentReq.IdempotencyKey); findErr == nil {
		resp := dto.CreatePaymentResponse{
			PaymentLinkID: existingAttempt.ProviderLinkID,
			PaymentURL:    existingAttempt.PaymentLink,
			ReferenceID:   existingAttempt.ProviderReferenceID,
		}
		log.Info("payment link retry success",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("payment_id", existingAttempt.PaymentID),
			zap.String("idempotency_key", paymentReq.IdempotencyKey),
			zap.Bool("idempotent_replay", true),
			zap.Bool("has_payment_url", resp.PaymentURL != ""),
		)
		return resp, nil
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		log.Error("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "idempotency_lookup"),
			zap.Error(findErr),
		)
		return dto.CreatePaymentResponse{}, findErr
	}

	payment, err := p.GetPaymentByInvoiceID(log, paymentReq.InvoiceID)
	if err != nil {
		return dto.CreatePaymentResponse{}, err
	}
	if payment.Source != constants.PaymentLink {
		log.Warn("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("source", payment.Source),
			zap.String("reason", "unsupported_payment_mode"),
		)
		return dto.CreatePaymentResponse{}, fmt.Errorf("retry link is only supported for payment_mode %s", constants.PaymentLink)
	}

	if latest, findErr := p.PaymentAttempt.FindByPaymentID(log, payment.ID); findErr == nil {
		if latest.PaymentStatus == constants.StatusPending {
			resp := dto.CreatePaymentResponse{
				PaymentLinkID: latest.ProviderLinkID,
				PaymentURL:    latest.PaymentLink,
				ReferenceID:   latest.ProviderReferenceID,
			}
			log.Info("payment link retry success",
				zap.String("invoice_id", paymentReq.InvoiceID),
				zap.String("payment_id", payment.ID),
				zap.String("reason", "reuse_pending_attempt"),
				zap.Bool("has_payment_url", resp.PaymentURL != ""),
			)
			return resp, nil
		}
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		log.Error("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "attempt_lookup"),
			zap.Error(findErr),
		)
		return dto.CreatePaymentResponse{}, findErr
	}

	provider, err := p.PaymentFactory.GetProvider(constants.ProviderNameRazorpay)
	if err != nil {
		log.Error("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "provider"),
			zap.Error(err),
		)
		return dto.CreatePaymentResponse{}, err
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()

	paymentResponse, err := provider.CreatePayment(ctx, paymentReq)
	if err != nil {
		log.Error("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "gateway_create"),
			zap.Error(err),
		)
		return dto.CreatePaymentResponse{}, err
	}

	tx := p.db.Begin()
	err = p.PaymentAttempt.CreateAttemptWithIdempotency(log, tx, payment.ID, paymentResponse, provider.Name(), paymentReq.IdempotencyKey)
	if err != nil {
		tx.Rollback()
		log.Error("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("payment_id", payment.ID),
			zap.String("reason", "db_attempt"),
			zap.Error(err),
		)
		return dto.CreatePaymentResponse{}, err
	}
	err = p.PrescriptionStatus.UpdateExtPrescriptionStatus(tx, paymentReq.PrescriptionID, constants.StatusPaymentPending)
	if err != nil {
		tx.Rollback()
		log.Error("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("prescription_id", paymentReq.PrescriptionID),
			zap.String("reason", "prescription_status"),
			zap.Error(err),
		)
		return dto.CreatePaymentResponse{}, err
	}
	if err = tx.Commit().Error; err != nil {
		log.Error("payment link retry failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "db_commit"),
			zap.Error(err),
		)
		return dto.CreatePaymentResponse{}, err
	}
	log.Info("payment link retry success",
		zap.String("payment_id", payment.ID),
		zap.String("invoice_id", paymentReq.InvoiceID),
		zap.String("provider_link_id", paymentResponse.PaymentLinkID),
		zap.String("idempotency_key", paymentReq.IdempotencyKey),
		zap.Bool("has_payment_url", paymentResponse.PaymentURL != ""),
	)
	return paymentResponse, nil
}

func (p *PaymentsService) CreatePendingPayment(log *zap.Logger, paymentReq dto.CreatePaymentCommand) (Payments, error) {
	log = ensureLog(log)
	if err := validateIdempotencyKey(paymentReq.IdempotencyKey); err != nil {
		log.Warn("payment pending create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("payment_mode", paymentReq.Source),
			zap.String("reason", "idempotency_required"),
		)
		return Payments{}, err
	}

	if existing, err := p.PaymentsRepository.FindByIdempotencyKey(log, paymentReq.IdempotencyKey); err == nil {
		log.Info("payment pending create success",
			zap.String("payment_id", existing.ID),
			zap.String("invoice_id", existing.InvoiceID),
			zap.String("payment_mode", existing.Source),
			zap.Bool("idempotent_replay", true),
		)
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("payment pending create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("payment_mode", paymentReq.Source),
			zap.String("reason", "idempotency_lookup"),
			zap.Error(err),
		)
		return Payments{}, err
	}

	tx := p.db.Begin()
	paymentModel := p.toPaymentModel(paymentReq)
	err := p.PaymentsRepository.Create(log, tx, paymentModel)
	if err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			existing, findErr := p.PaymentsRepository.FindByIdempotencyKey(log, paymentReq.IdempotencyKey)
			if findErr == nil {
				log.Info("payment pending create success",
					zap.String("payment_id", existing.ID),
					zap.String("invoice_id", existing.InvoiceID),
					zap.String("payment_mode", existing.Source),
					zap.Bool("idempotent_replay", true),
				)
				return existing, nil
			}
		}
		log.Error("payment pending create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("payment_mode", paymentReq.Source),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return Payments{}, err
	}
	if paymentReq.PrescriptionID != "" {
		if err = p.PrescriptionStatus.UpdateExtPrescriptionStatus(tx, paymentReq.PrescriptionID, constants.StatusPaymentPending); err != nil {
			tx.Rollback()
			log.Error("payment pending create failed",
				zap.String("invoice_id", paymentReq.InvoiceID),
				zap.String("prescription_id", paymentReq.PrescriptionID),
				zap.String("reason", "prescription_status"),
				zap.Error(err),
			)
			return Payments{}, err
		}
	}
	if err = tx.Commit().Error; err != nil {
		log.Error("payment pending create failed",
			zap.String("invoice_id", paymentReq.InvoiceID),
			zap.String("reason", "db_commit"),
			zap.Error(err),
		)
		return Payments{}, err
	}
	log.Info("payment pending create success",
		zap.String("payment_id", paymentModel.ID),
		zap.String("invoice_id", paymentReq.InvoiceID),
		zap.String("payment_mode", paymentReq.Source),
	)
	return paymentModel, nil
}

func (p *PaymentsService) ConfirmManualPayment(log *zap.Logger, invoiceID, organisationID, paymentMode, txnRef string) error {
	log = ensureLog(log)
	_ = txnRef
	if invoiceID == "" {
		log.Warn("payment confirm failed",
			zap.String("reason", "missing_invoice_id"),
		)
		return wrapError.ErrInvalidRequest
	}
	if organisationID == "" {
		log.Warn("payment confirm failed",
			zap.String("invoice_id", invoiceID),
			zap.String("reason", "missing_organisation_id"),
		)
		return wrapError.ErrInvalidRequest
	}
	if paymentMode != constants.PaymentCash && paymentMode != constants.PaymentQR {
		log.Warn("payment confirm failed",
			zap.String("invoice_id", invoiceID),
			zap.String("organisation_id", organisationID),
			zap.String("payment_mode", paymentMode),
			zap.String("reason", "unsupported_payment_mode"),
		)
		return wrapError.ErrUnsupportedPaymentMode
	}

	payment, err := p.GetPaymentByInvoiceIDAndOrganisationID(log, invoiceID, organisationID)
	if err != nil {
		if errors.Is(err, wrapError.ErrPaymentNotFound) {
			log.Warn("payment confirm failed",
				zap.String("invoice_id", invoiceID),
				zap.String("organisation_id", organisationID),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrPaymentNotFound
		}
		log.Error("payment confirm failed",
			zap.String("invoice_id", invoiceID),
			zap.String("organisation_id", organisationID),
			zap.String("reason", "payment_lookup"),
			zap.Error(err),
		)
		return wrapError.ErrPaymentConfirmFailed
	}
	if payment.Source != constants.PaymentCash && payment.Source != constants.PaymentQR {
		log.Warn("payment confirm failed",
			zap.String("invoice_id", invoiceID),
			zap.String("organisation_id", organisationID),
			zap.String("payment_mode", paymentMode),
			zap.String("source", payment.Source),
			zap.String("reason", "source_mismatch"),
		)
		return wrapError.ErrInvalidRequest
	}

	tx := p.db.Begin()
	if paymentMode != payment.Source {
		channel := constants.PaymentCash
		if paymentMode == constants.PaymentQR {
			channel = constants.PaymentUPI
		}
		if err = p.PaymentsRepository.UpdateSource(log, tx, payment.ID, paymentMode, channel); err != nil {
			tx.Rollback()
			log.Error("payment confirm failed",
				zap.String("invoice_id", invoiceID),
				zap.String("organisation_id", organisationID),
				zap.String("payment_id", payment.ID),
				zap.String("payment_mode", paymentMode),
				zap.String("source", payment.Source),
				zap.String("reason", "source_update"),
				zap.Error(err),
			)
			return wrapError.ErrPaymentConfirmFailed
		}
		log.Info("payment source updated on confirm",
			zap.String("invoice_id", invoiceID),
			zap.String("organisation_id", organisationID),
			zap.String("payment_id", payment.ID),
			zap.String("from_source", payment.Source),
			zap.String("to_source", paymentMode),
		)
		payment.Source = paymentMode
		payment.Channel = channel
	}
	err = p.FulfillmentSvc.FulfillPaidInvoice(log, tx, invoiceID)
	if err != nil {
		tx.Rollback()
		log.Error("payment confirm failed",
			zap.String("invoice_id", invoiceID),
			zap.String("organisation_id", organisationID),
			zap.String("reason", "fulfillment_failed"),
			zap.Error(err),
		)
		return wrapError.ErrPaymentConfirmFailed
	}
	if err = tx.Commit().Error; err != nil {
		log.Error("payment confirm failed",
			zap.String("invoice_id", invoiceID),
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_commit"),
			zap.Error(err),
		)
		return wrapError.ErrPaymentConfirmFailed
	}

	paidAt := time.Now()
	notifyErr := p.FulfillmentSvc.NotifyPaymentReceived(log, payment, paidAt)
	notificationEnqueued := notifyErr == nil
	if notifyErr != nil {
		log.Error("payment notification failed",
			zap.String("invoice_id", invoiceID),
			zap.String("organisation_id", organisationID),
			zap.String("payment_id", payment.ID),
			zap.String("reason", "notification"),
			zap.Error(notifyErr),
		)
	}
	log.Info("payment confirm success",
		zap.String("invoice_id", invoiceID),
		zap.String("organisation_id", organisationID),
		zap.String("payment_id", payment.ID),
		zap.String("payment_mode", paymentMode),
		zap.Bool("notification_enqueued", notificationEnqueued),
	)
	return nil
}

func (p *PaymentsService) GetPaymentByIdempotencyKey(log *zap.Logger, key string) (Payments, error) {
	log = ensureLog(log)
	if err := validateIdempotencyKey(key); err != nil {
		return Payments{}, err
	}
	return p.PaymentsRepository.FindByIdempotencyKey(log, key)
}

func (p *PaymentsService) GetPaymentURLByPaymentID(log *zap.Logger, paymentID string) (string, error) {
	log = ensureLog(log)
	attempt, err := p.PaymentAttempt.FindByPaymentID(log, paymentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return attempt.PaymentLink, nil
}

func (p *PaymentsService) responseFromExistingPayment(log *zap.Logger, payment Payments) (dto.CreatePaymentResponse, error) {
	attempt, err := p.PaymentAttempt.FindByPaymentID(log, payment.ID)
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

func (p *PaymentsService) FindInvoiceByPaymentAttempt(log *zap.Logger, query string, args ...interface{}) (Payments, error) {
	return p.PaymentsRepository.FindInvoiceByPaymentAttempt(ensureLog(log), query, args...)
}

func (p *PaymentsService) GetPaymentByInvoiceID(log *zap.Logger, invoiceID string) (Payments, error) {
	log = ensureLog(log)
	if invoiceID == "" {
		return Payments{}, wrapError.ErrInvalidRequest
	}
	payment, err := p.PaymentsRepository.FindByInvoiceID(log, invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Payments{}, wrapError.ErrPaymentNotFound
		}
		return Payments{}, err
	}
	return payment, nil
}

func (p *PaymentsService) GetPaymentByInvoiceIDAndOrganisationID(log *zap.Logger, invoiceID, organisationID string) (Payments, error) {
	log = ensureLog(log)
	if invoiceID == "" || organisationID == "" {
		return Payments{}, wrapError.ErrInvalidRequest
	}
	query := `
		SELECT payments.*
		FROM payments
		JOIN invoices ON invoices.id = payments.invoice_id
		WHERE payments.invoice_id = ?
		  AND invoices.organisation_id = ?
		ORDER BY payments.created_at DESC
		LIMIT 1
	`
	payment, err := p.PaymentsRepository.FindInvoiceByPaymentAttempt(log, query, invoiceID, organisationID)
	if err != nil {
		return Payments{}, err
	}
	// Raw().Scan() returns nil error with zero rows — treat empty ID as not found.
	if payment.ID == "" {
		return Payments{}, wrapError.ErrPaymentNotFound
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
