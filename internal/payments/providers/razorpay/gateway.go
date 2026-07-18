package razorpay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"time"

	"hospital-backend/internal/payments/dto"
	"hospital-backend/pkg/constants"
)

type gateway struct {
	client *Config
}

func NewGateway(razorpayClient *RazorpayConfig, callbackUrl string, webhooksecret string) *gateway {
	return &gateway{client: &Config{
		PaymentConfig: PaymentConfig{
			CallbackUrl:    callbackUrl,
			RazorpayConfig: *razorpayClient,
			WebhookSecret:  webhooksecret,
		},
	}}
}
func (g *gateway) Name() string {
	return constants.ProviderNameRazorpay
}

func (g *gateway) CreatePayment(ctx context.Context, req dto.CreatePaymentCommand) (dto.CreatePaymentResponse, error) {
	createPaymentLinkRequest := g.toCreatePaymentLinkRequest(req)
	reqPayload := g.mapExternalReq(createPaymentLinkRequest)
	paymentResponse, err := g.client.PaymentConfig.RazorpayConfig.CreatePaymentLink(ctx, createPaymentLinkRequest)
	if err != nil {
		return dto.CreatePaymentResponse{}, err
	}
	paymentResponse.RequestPayload = reqPayload
	return paymentResponse, nil
}
func (g *gateway) mapExternalReq(paymentLinkPayload createPaymentLinkRequest) map[string]interface{} {
	paymentPayload := map[string]interface{}{
		"amount":         paymentLinkPayload.Amount,
		"currency":       paymentLinkPayload.Currency,
		"accept_partial": paymentLinkPayload.AcceptPartial,
		"reference_id":   paymentLinkPayload.ReferenceID,
		"description":    paymentLinkPayload.Description,
		"customer": map[string]interface{}{
			"name":    paymentLinkPayload.Customer.Name,
			"email":   paymentLinkPayload.Customer.Email,
			"contact": paymentLinkPayload.Customer.Contact,
		},
		"notify": map[string]interface{}{
			"email": paymentLinkPayload.Notify.Email,
		},
		"reminder_enable": paymentLinkPayload.ReminderEnable,
		"notes":           paymentLinkPayload.Notes,
		"callback_url":    paymentLinkPayload.CallbackURL,
		"callback_method": paymentLinkPayload.CallbackMethod,
	}
	if paymentLinkPayload.ExpireBy > 0 {
		paymentPayload["expire_by"] = paymentLinkPayload.ExpireBy
	}
	return paymentPayload
}

func (g *gateway) toCreatePaymentLinkRequest(req dto.CreatePaymentCommand) createPaymentLinkRequest {
	var expireBy int64
	if !req.ExpiresAt.IsZero() {
		expireBy = req.ExpiresAt.Unix()
	}
	return createPaymentLinkRequest{
		// Razorpay expects amount in paise (₹500 → 50000)
		Amount:   int64(math.Round(req.Amount * 100)),
		Currency: req.Currency,
		//UPILink:       true,
		AcceptPartial: false,
		ExpireBy:      expireBy,
		ReferenceID:   req.ReferenceID,
		Description:   req.Description,
		Customer: customer{
			Name:    req.Customer.Name,
			Email:   req.Customer.Email,
			Contact: req.Customer.Mobile,
		},
		Notify: notify{
			Email: req.SendEmail,
		},
		ReminderEnable: req.SendSMS,
		Notes:          req.Metadata,
		CallbackURL:    g.client.PaymentConfig.CallbackUrl,
		CallbackMethod: "get", // payment links only allow get
	}
}
func (g *gateway) VerifySignature(payload []byte, signature string) (bool, error) {
	if signature == "" {
		return false, errors.New("missing razorpay signature")
	}
	mac := hmac.New(sha256.New, []byte(g.client.PaymentConfig.WebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return false, errors.New("payment verification failed")
	}
	return true, nil
}
func (g *gateway) ParseWebhookEvent(payload []byte) (dto.ParsedWebhookEvent, error) {
	var webhookEvent WebhookEvent
	err := json.Unmarshal(payload, &webhookEvent)
	if err != nil {
		return dto.ParsedWebhookEvent{}, err
	}
	dtowebhookevent := g.toParsedWebhookEvent(webhookEvent, payload)
	return dtowebhookevent, nil
}
func (g *gateway) toParsedWebhookEvent(webhookEvent WebhookEvent, payload []byte) dto.ParsedWebhookEvent {
	payment := webhookEvent.Payload.Payment.Entity
	paymentLink := webhookEvent.Payload.PaymentLink.Entity
	order := webhookEvent.Payload.Order.Entity

	event := dto.ParsedWebhookEvent{
		EventType:         webhookEvent.Event,
		ProviderEventID:   webhookEvent.AccountID,
		ProviderLinkID:    paymentLink.ID,
		ProviderOrderID:   order.ID,
		ProviderPaymentID: payment.ID,
		ReferenceID:       paymentLink.ReferenceID,
		AmountPaid:        float64(payment.Amount) / 100, // paise → rupees
		AmountTransferred: float64(payment.AmountTransferred) / 100,
		PaymentStatus:     payment.Status,
		PaymentLinkStatus: paymentLink.Status,
		AcceptPartial:     paymentLink.AcceptPartial,
		RawPayload:        payload,
	}

	if payment.CreatedAt > 0 {
		event.PaidAt = time.Unix(payment.CreatedAt, 0)
	}
	if payment.UPI != nil {
		event.PayerVPA = payment.UPI.VPA
		event.PayerAccountType = payment.UPI.PayerAccountType
	}
	if payment.ErrorDescription != nil {
		event.PaymentError = payment.ErrorDescription
	}
	if payment.ErrorCode != nil {
		event.PaymentErrorCode = payment.ErrorCode
	}
	return event
}
