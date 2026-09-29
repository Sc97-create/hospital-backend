package payments

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"

	dto "hospital-backend/central/payments/dto"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

const (
	CurrencyINR     = "INR"
	createOrderPath = "/api/v1/hospital/internal/payment/createOrder"
)

var allowedCurrencies = map[string]struct{}{
	CurrencyINR: {},
}

// OrderAPI is the shared internal HTTP client.
type OrderAPI interface {
	Post(ctx context.Context, path string, payload any) (int, []byte, error)
}

// PaymentService creates provider orders for central subscriptions.
type PaymentService struct {
	API OrderAPI
}

func NewPaymentService(api OrderAPI) *PaymentService {
	return &PaymentService{API: api}
}

// AmountInPaise converts major-unit amount to Razorpay's smallest currency unit.
func AmountInPaise(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// CreateOrder validates the payload and calls the internal createOrder API.
func (s *PaymentService) CreateOrder(log *zap.Logger, ctx context.Context, req dto.CreateOrderRequest) (dto.CreateOrderResponse, error) {
	log = ensureLog(log)
	req = normalizeCreateOrderRequest(req)
	if err := validateCreateOrder(log, req); err != nil {
		return dto.CreateOrderResponse{}, err
	}
	if s == nil || s.API == nil {
		log.Error("payment order api call failed", zap.String("reason", "client_missing"))
		return dto.CreateOrderResponse{}, wrapError.ErrOrderCreateFailed
	}
	status, body, err := s.API.Post(ctx, createOrderPath, req)
	if err != nil {
		log.Error("payment order api call failed", zap.String("reason", "http_do"), zap.Error(err))
		return dto.CreateOrderResponse{}, wrapError.ErrOrderCreateFailed
	}
	return decodeCreateOrder(log, status, body)
}

func decodeCreateOrder(log *zap.Logger, status int, body []byte) (dto.CreateOrderResponse, error) {
	if status == http.StatusUnauthorized {
		log.Warn("payment order api call failed", zap.String("reason", "unauthorized"), zap.Int("status", status))
		return dto.CreateOrderResponse{}, wrapError.ErrUnauthorized
	}
	if status == http.StatusBadRequest {
		log.Warn("payment order api call failed", zap.String("reason", "bad_request"), zap.Int("status", status), zap.ByteString("body", body))
		return dto.CreateOrderResponse{}, mapBadRequestError(body)
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		log.Error("payment order api call failed", zap.String("reason", "non_2xx"), zap.Int("status", status), zap.ByteString("body", body))
		return dto.CreateOrderResponse{}, wrapError.ErrOrderCreateFailed
	}
	var out dto.CreateOrderResponse
	if err := json.Unmarshal(body, &out); err != nil || strings.TrimSpace(out.ID) == "" {
		log.Error("payment order api call failed", zap.String("reason", "decode_response"), zap.ByteString("body", body), zap.Error(err))
		return dto.CreateOrderResponse{}, wrapError.ErrOrderCreateFailed
	}
	log.Info("payment order api call success", zap.String("order_id", out.ID), zap.String("currency", out.Currency), zap.Int64("amount", out.Amount))
	return out, nil
}

func validateCreateOrder(log *zap.Logger, req dto.CreateOrderRequest) error {
	if req.Amount <= 0 {
		log.Warn("payment order validation failed", zap.String("reason", "invalid_amount"), zap.Int64("amount", req.Amount))
		return wrapError.ErrInvalidAmount
	}
	if req.Currency == "" {
		log.Warn("payment order validation failed", zap.String("reason", "missing_currency"))
		return wrapError.ErrInvalidCurrency
	}
	if _, ok := allowedCurrencies[req.Currency]; !ok {
		log.Warn("payment order validation failed", zap.String("reason", "unsupported_currency"), zap.String("currency", req.Currency))
		return wrapError.ErrInvalidCurrency
	}
	if len(req.Receipt) > 40 {
		log.Warn("payment order validation failed", zap.String("reason", "receipt_too_long"))
		return wrapError.ErrInvalidRequest
	}
	return nil
}

func normalizeCreateOrderRequest(req dto.CreateOrderRequest) dto.CreateOrderRequest {
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.Receipt = strings.TrimSpace(req.Receipt)
	return req
}

func mapBadRequestError(body []byte) error {
	var errBody dto.APIErrorBody
	_ = json.Unmarshal(body, &errBody)
	msg := strings.ToLower(strings.TrimSpace(errBody.Error + " " + errBody.Message))
	switch {
	case strings.Contains(msg, "amount"):
		return wrapError.ErrInvalidAmount
	case strings.Contains(msg, "currency"):
		return wrapError.ErrInvalidCurrency
	default:
		return wrapError.ErrInvalidRequest
	}
}

func ensureLog(log *zap.Logger) *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}
	return log
}
