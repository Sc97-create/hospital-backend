package payments

import (
	"strings"
	"time"

	"hospital-backend/internal/payments/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const subscriptionProvider = "razorpay"

// RecordSubscriptionPayment stores the Razorpay order on a payment and a matching attempt.
// source is link, channel is card, and customer_id is required.
func (p *PaymentsService) RecordSubscriptionPayment(log *zap.Logger, req dto.SubscriptionPaymentRequest) (dto.SubscriptionPaymentResponse, error) {
	log = ensureLog(log)
	req = normalizeSubscriptionPayment(req)
	if err := validateSubscriptionPayment(log, req); err != nil {
		return dto.SubscriptionPaymentResponse{}, err
	}
	if p == nil || p.db == nil {
		log.Error("subscription payment failed", zap.String("reason", "db_missing"))
		return dto.SubscriptionPaymentResponse{}, wrapError.ErrPaymentCreateFailed
	}
	if existing, err := p.existingSubscriptionPayment(log, req.OrderID); err != nil || existing.PaymentID != "" {
		return existing, err
	}
	return p.insertSubscriptionPayment(log, req)
}

func (p *PaymentsService) ConfirmSubscriptionPayment(log *zap.Logger, req dto.ConfirmSubscriptionPaymentRequest) (dto.ConfirmSubscriptionPaymentResponse, error) {
	log = ensureLog(log)
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.ProviderPaymentID = strings.TrimSpace(req.ProviderPaymentID)
	req.PaymentSignature = strings.TrimSpace(req.PaymentSignature)
	if req.OrderID == "" || req.ProviderPaymentID == "" || req.PaymentSignature == "" {
		log.Warn("subscription payment confirm failed", zap.String("reason", "missing_fields"))
		return dto.ConfirmSubscriptionPaymentResponse{}, wrapError.ErrInvalidRequest
	}
	if p == nil || p.db == nil {
		log.Error("subscription payment confirm failed", zap.String("reason", "db_missing"))
		return dto.ConfirmSubscriptionPaymentResponse{}, wrapError.ErrPaymentConfirmFailed
	}

	paymentID, subscriptionID, err := p.subscriptionPaymentRef(log, req.OrderID)
	if err != nil {
		return dto.ConfirmSubscriptionPaymentResponse{}, err
	}
	if err := p.storeSubscriptionConfirmation(log, req); err != nil {
		return dto.ConfirmSubscriptionPaymentResponse{}, err
	}
	log.Info("subscription payment confirm success",
		zap.String("payment_id", paymentID),
		zap.String("order_id", req.OrderID),
		zap.String("subscription_id", subscriptionID),
	)
	return dto.ConfirmSubscriptionPaymentResponse{
		PaymentID:      paymentID,
		OrderID:        req.OrderID,
		SubscriptionID: subscriptionID,
	}, nil
}

func normalizeSubscriptionPayment(req dto.SubscriptionPaymentRequest) dto.SubscriptionPaymentRequest {
	req.InvoiceID = strings.TrimSpace(req.InvoiceID)
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Currency == "" {
		req.Currency = constants.IndCurrnecy
	}
	return req
}

func validateSubscriptionPayment(log *zap.Logger, req dto.SubscriptionPaymentRequest) error {
	if _, err := uuid.Parse(req.InvoiceID); err != nil {
		log.Warn("subscription payment failed", zap.String("reason", "invalid_invoice_id"))
		return wrapError.ErrInvalidRequest
	}
	if _, err := uuid.Parse(req.CustomerID); err != nil {
		log.Warn("subscription payment failed", zap.String("reason", "invalid_customer_id"))
		return wrapError.ErrInvalidRequest
	}
	if req.OrderID == "" {
		log.Warn("subscription payment failed", zap.String("reason", "missing_order_id"))
		return wrapError.ErrInvalidRequest
	}
	if req.Amount <= 0 {
		log.Warn("subscription payment failed", zap.String("reason", "invalid_amount"))
		return wrapError.ErrInvalidAmount
	}
	if req.Currency != constants.IndCurrnecy {
		log.Warn("subscription payment failed", zap.String("reason", "invalid_currency"))
		return wrapError.ErrInvalidCurrency
	}
	return nil
}

func (p *PaymentsService) existingSubscriptionPayment(log *zap.Logger, orderID string) (dto.SubscriptionPaymentResponse, error) {
	var row Payments
	err := p.db.Where("order_id = ?", orderID).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.SubscriptionPaymentResponse{}, nil
		}
		log.Error("subscription payment failed", zap.String("reason", "lookup"), zap.Error(err))
		return dto.SubscriptionPaymentResponse{}, wrapError.ErrPaymentCreateFailed
	}
	return dto.SubscriptionPaymentResponse{PaymentID: row.ID, OrderID: row.OrderID}, nil
}

func (p *PaymentsService) insertSubscriptionPayment(log *zap.Logger, req dto.SubscriptionPaymentRequest) (dto.SubscriptionPaymentResponse, error) {
	now := time.Now()
	paymentID := uuid.NewString()
	customerID := req.CustomerID
	row := Payments{
		ID:             paymentID,
		InvoiceID:      req.InvoiceID,
		CustomerID:     &customerID,
		OrderID:        req.OrderID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Source:         constants.PaymentLink,
		Channel:        constants.PaymentCard,
		InitiatedBy:    req.CustomerID,
		IdempotencyKey: "subscription:" + req.OrderID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	attempt := PaymentAttempts{
		ID:              uuid.NewString(),
		PaymentID:       paymentID,
		Provider:        subscriptionProvider,
		AttemptNo:       1,
		ProviderLinkID:  req.OrderID,
		ProviderOrderID: req.OrderID,
		PaymentStatus:   constants.StatusPending,
		CreatedAt:       now,
	}
	tx := p.db.Begin()
	if tx.Error != nil {
		log.Error("subscription payment failed", zap.String("reason", "begin"), zap.Error(tx.Error))
		return dto.SubscriptionPaymentResponse{}, wrapError.ErrPaymentCreateFailed
	}
	if err := tx.Omit("PatientID").Create(&row).Error; err != nil {
		tx.Rollback()
		log.Error("subscription payment failed", zap.String("order_id", req.OrderID), zap.String("reason", "db_create"), zap.Error(err))
		return dto.SubscriptionPaymentResponse{}, wrapError.ErrPaymentCreateFailed
	}
	if err := tx.Omit("PaidAt").Create(&attempt).Error; err != nil {
		tx.Rollback()
		log.Error("subscription payment failed", zap.String("order_id", req.OrderID), zap.String("reason", "attempt_create"), zap.Error(err))
		return dto.SubscriptionPaymentResponse{}, wrapError.ErrPaymentCreateFailed
	}
	if err := tx.Commit().Error; err != nil {
		log.Error("subscription payment failed", zap.String("order_id", req.OrderID), zap.String("reason", "commit"), zap.Error(err))
		return dto.SubscriptionPaymentResponse{}, wrapError.ErrPaymentCreateFailed
	}
	log.Info("subscription payment success",
		zap.String("payment_id", paymentID),
		zap.String("invoice_id", req.InvoiceID),
		zap.String("order_id", req.OrderID),
		zap.String("customer_id", req.CustomerID),
		zap.String("source", constants.PaymentLink),
		zap.String("channel", constants.PaymentCard),
	)
	return dto.SubscriptionPaymentResponse{PaymentID: paymentID, OrderID: req.OrderID}, nil
}

func (p *PaymentsService) subscriptionPaymentRef(log *zap.Logger, orderID string) (string, string, error) {
	var row struct {
		PaymentID      string `gorm:"column:payment_id"`
		SubscriptionID string `gorm:"column:subscription_id"`
	}
	err := p.db.Raw(`
		SELECT p.id AS payment_id, i.billing_id AS subscription_id
		FROM payments p
		JOIN invoices i ON i.id = p.invoice_id
		WHERE p.order_id = ?
		LIMIT 1
	`, orderID).Scan(&row).Error
	if err != nil {
		log.Error("subscription payment confirm failed", zap.String("order_id", orderID), zap.String("reason", "lookup"), zap.Error(err))
		return "", "", wrapError.ErrPaymentConfirmFailed
	}
	if row.PaymentID == "" {
		log.Warn("subscription payment confirm failed", zap.String("order_id", orderID), zap.String("reason", "not_found"))
		return "", "", wrapError.ErrPaymentNotFound
	}
	return row.PaymentID, row.SubscriptionID, nil
}

func (p *PaymentsService) storeSubscriptionConfirmation(log *zap.Logger, req dto.ConfirmSubscriptionPaymentRequest) error {
	now := time.Now()
	err := p.db.Model(&Payments{}).Where("order_id = ?", req.OrderID).Updates(map[string]any{
		"provider_payment_id": req.ProviderPaymentID,
		"payment_signature":   req.PaymentSignature,
		"updated_at":          now,
	}).Error
	if err != nil {
		log.Error("subscription payment confirm failed", zap.String("order_id", req.OrderID), zap.String("reason", "payment_update"), zap.Error(err))
		return wrapError.ErrPaymentConfirmFailed
	}
	err = p.db.Model(&PaymentAttempts{}).Where("provider_order_id = ?", req.OrderID).Updates(map[string]any{
		"provider_payment_id": req.ProviderPaymentID,
		"payment_signature":   req.PaymentSignature,
	}).Error
	if err != nil {
		log.Error("subscription payment confirm failed", zap.String("order_id", req.OrderID), zap.String("reason", "attempt_update"), zap.Error(err))
		return wrapError.ErrPaymentConfirmFailed
	}
	return nil
}
