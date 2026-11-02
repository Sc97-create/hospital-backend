package billing

import (
	"strings"
	"time"

	"hospital-backend/internal/billing/dto"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CreateSubscriptionInvoice writes one invoice for a subscription.
// subscription_new is unique per subscription. A second unpaid subscription_renewal is rejected.
func (s *InvoiceServ) CreateSubscriptionInvoice(log *zap.Logger, req dto.SubscriptionInvoiceRequest) (dto.SubscriptionInvoiceResponse, error) {
	log = ensureLog(log)
	req.SubscriptionID = strings.TrimSpace(req.SubscriptionID)
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	if err := validateSubscriptionInvoice(log, req); err != nil {
		return dto.SubscriptionInvoiceResponse{}, err
	}
	if s == nil || s.db == nil {
		log.Error("subscription invoice failed", zap.String("reason", "db_missing"))
		return dto.SubscriptionInvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
	}

	paymentType, err := s.subscriptionPaymentType(log, req.Renewal, req.SubscriptionID)
	if err != nil {
		return dto.SubscriptionInvoiceResponse{}, err
	}
	if err := s.rejectDuplicateSubscriptionInvoice(log, paymentType, req.SubscriptionID); err != nil {
		return dto.SubscriptionInvoiceResponse{}, err
	}

	row, err := s.insertSubscriptionInvoice(log, req, paymentType)
	if err != nil {
		return dto.SubscriptionInvoiceResponse{}, err
	}
	log.Info("subscription invoice success",
		zap.String("invoice_id", row.ID),
		zap.String("subscription_id", req.SubscriptionID),
		zap.String("payment_type", string(paymentType)),
		zap.String("customer_id", req.CustomerID),
	)
	return dto.SubscriptionInvoiceResponse{
		InvoiceID:   row.ID,
		PaymentType: string(paymentType),
		Status:      row.Status,
	}, nil
}

func (s *InvoiceServ) insertSubscriptionInvoice(log *zap.Logger, req dto.SubscriptionInvoiceRequest, paymentType PaymentType) (Invoice, error) {
	now := time.Now()
	customerID := req.CustomerID
	row := Invoice{
		ID:             uuid.NewString(),
		InvoiceCode:    s.createCode(),
		PaymentType:    paymentType,
		BillingID:      req.SubscriptionID,
		CustomerID:     &customerID,
		Status:         StatusUnpaid,
		OrganisationID: req.OrganisationID,
		SubtotalAmount: req.Amount,
		TotalAmount:    req.Amount,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	// Patient and cashier stay null. Hospital checkout still sets both on Invoice.
	if err := s.db.Omit("PatientID", "CashierID").Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			log.Warn("subscription invoice failed",
				zap.String("subscription_id", req.SubscriptionID),
				zap.String("payment_type", string(paymentType)),
				zap.String("reason", "already_exists"),
			)
			return Invoice{}, wrapError.ErrInvoiceAlreadyExists
		}
		log.Error("subscription invoice failed",
			zap.String("subscription_id", req.SubscriptionID),
			zap.String("payment_type", string(paymentType)),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return Invoice{}, wrapError.ErrInvoiceCreateFailed
	}
	return row, nil
}

func validateSubscriptionInvoice(log *zap.Logger, req dto.SubscriptionInvoiceRequest) error {
	if _, err := uuid.Parse(req.SubscriptionID); err != nil {
		log.Warn("subscription invoice failed", zap.String("reason", "invalid_subscription_id"))
		return wrapError.ErrInvalidRequest
	}
	if _, err := uuid.Parse(req.CustomerID); err != nil {
		log.Warn("subscription invoice failed", zap.String("reason", "invalid_customer_id"))
		return wrapError.ErrInvalidRequest
	}
	if _, err := uuid.Parse(req.OrganisationID); err != nil {
		log.Warn("subscription invoice failed", zap.String("reason", "invalid_organisation_id"))
		return wrapError.ErrInvalidRequest
	}
	if req.Amount <= 0 {
		log.Warn("subscription invoice failed", zap.String("reason", "invalid_amount"))
		return wrapError.ErrInvalidAmount
	}
	return nil
}

func (s *InvoiceServ) subscriptionPaymentType(log *zap.Logger, renewal bool, subscriptionID string) (PaymentType, error) {
	if !renewal {
		return PaymentTypeSubscriptionNew, nil
	}
	var count int64
	err := s.db.Model(&Invoice{}).
		Where("billing_id = ? AND payment_type IN ?", subscriptionID, []PaymentType{PaymentTypeSubscriptionNew, PaymentTypeSubscriptionRenewal}).
		Count(&count).Error
	if err != nil {
		log.Error("subscription invoice failed",
			zap.String("subscription_id", subscriptionID),
			zap.String("reason", "type_lookup"),
			zap.Error(err),
		)
		return "", wrapError.ErrInvoiceCreateFailed
	}
	if count == 0 {
		return PaymentTypeSubscriptionNew, nil
	}
	return PaymentTypeSubscriptionRenewal, nil
}

func (s *InvoiceServ) rejectDuplicateSubscriptionInvoice(log *zap.Logger, paymentType PaymentType, subscriptionID string) error {
	var count int64
	query := s.db.Model(&Invoice{}).Where("payment_type = ? AND billing_id = ?", paymentType, subscriptionID)
	if paymentType == PaymentTypeSubscriptionRenewal {
		query = query.Where("status = ?", StatusUnpaid)
	}
	if err := query.Count(&count).Error; err != nil {
		log.Error("subscription invoice failed",
			zap.String("subscription_id", subscriptionID),
			zap.String("reason", "duplicate_lookup"),
			zap.Error(err),
		)
		return wrapError.ErrInvoiceCreateFailed
	}
	if count > 0 {
		log.Warn("subscription invoice failed",
			zap.String("subscription_id", subscriptionID),
			zap.String("payment_type", string(paymentType)),
			zap.String("reason", "already_exists"),
		)
		return wrapError.ErrInvoiceAlreadyExists
	}
	return nil
}
