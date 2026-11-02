package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	dto "hospital-backend/central/payments/dto"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

const (
	subscriptionInvoicePath        = "/api/v1/hospital/internal/invoice/subscription"
	subscriptionPaymentPath        = "/api/v1/hospital/internal/payment/subscription"
	subscriptionPaymentConfirmPath = "/api/v1/hospital/internal/payment/subscription/confirm"
)

// CreateSubscriptionInvoice calls the hospital invoice API.
func (s *PaymentService) CreateSubscriptionInvoice(log *zap.Logger, ctx context.Context, req dto.SubscriptionInvoiceRequest) (dto.SubscriptionInvoiceResponse, error) {
	log = ensureLog(log)
	var out dto.SubscriptionInvoiceResponse
	if err := s.postInternal(log, ctx, subscriptionInvoicePath, req, &out, wrapError.ErrInvoiceCreateFailed); err != nil {
		return dto.SubscriptionInvoiceResponse{}, err
	}
	if strings.TrimSpace(out.InvoiceID) == "" {
		log.Error("subscription invoice api call failed", zap.String("reason", "empty_invoice_id"))
		return dto.SubscriptionInvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
	}
	return out, nil
}

// RecordSubscriptionPayment calls the hospital payment API.
func (s *PaymentService) RecordSubscriptionPayment(log *zap.Logger, ctx context.Context, req dto.SubscriptionPaymentRequest) error {
	log = ensureLog(log)
	var out dto.SubscriptionPaymentResponse
	if err := s.postInternal(log, ctx, subscriptionPaymentPath, req, &out, wrapError.ErrPaymentCreateFailed); err != nil {
		return err
	}
	if strings.TrimSpace(out.PaymentID) == "" {
		log.Error("subscription payment api call failed", zap.String("reason", "empty_payment_id"))
		return wrapError.ErrPaymentCreateFailed
	}
	return nil
}

// ConfirmSubscriptionPayment stores the Checkout ids on the payment via the hospital API.
func (s *PaymentService) ConfirmSubscriptionPayment(log *zap.Logger, ctx context.Context, req dto.ConfirmSubscriptionPaymentRequest) (dto.ConfirmSubscriptionPaymentResponse, error) {
	log = ensureLog(log)
	var out dto.ConfirmSubscriptionPaymentResponse
	if err := s.postInternal(log, ctx, subscriptionPaymentConfirmPath, req, &out, wrapError.ErrPaymentConfirmFailed); err != nil {
		return dto.ConfirmSubscriptionPaymentResponse{}, err
	}
	if strings.TrimSpace(out.SubscriptionID) == "" {
		log.Error("subscription payment confirm api call failed", zap.String("reason", "empty_subscription_id"))
		return dto.ConfirmSubscriptionPaymentResponse{}, wrapError.ErrPaymentConfirmFailed
	}
	return out, nil
}

func (s *PaymentService) postInternal(log *zap.Logger, ctx context.Context, path string, payload any, out any, fallback error) error {
	if s == nil || s.API == nil {
		log.Error("internal api call failed", zap.String("reason", "client_missing"), zap.String("path", path))
		return fallback
	}
	if ctx == nil {
		ctx = context.Background()
	}
	status, body, err := s.API.Post(ctx, path, payload)
	if err != nil {
		log.Error("internal api call failed", zap.String("reason", "http_do"), zap.String("path", path), zap.Error(err))
		return fallback
	}
	if status == http.StatusUnauthorized {
		log.Warn("internal api call failed", zap.String("reason", "unauthorized"), zap.String("path", path))
		return wrapError.ErrUnauthorized
	}
	if status == http.StatusBadRequest {
		log.Warn("internal api call failed", zap.String("reason", "bad_request"), zap.String("path", path))
		return wrapError.ErrInvalidRequest
	}
	if status == http.StatusConflict {
		log.Warn("internal api call failed", zap.String("reason", "conflict"), zap.String("path", path))
		return wrapError.ErrInvoiceAlreadyExists
	}
	if status == http.StatusNotFound {
		log.Warn("internal api call failed", zap.String("reason", "not_found"), zap.String("path", path))
		return wrapError.ErrPaymentNotFound
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		log.Error("internal api call failed", zap.String("reason", "non_2xx"), zap.String("path", path), zap.Int("status", status))
		return fallback
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		log.Error("internal api call failed", zap.String("reason", "decode_response"), zap.String("path", path), zap.Error(err))
		return fallback
	}
	return nil
}
