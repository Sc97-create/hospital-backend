package razorpay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"hospital-backend/internal/payments"
	"hospital-backend/internal/payments/dto"
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
	return payments.ProviderNameRazorpay
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
		"amount":                   paymentLinkPayload.Amount,
		"currency":                 paymentLinkPayload.Currency,
		"accept_partial":           paymentLinkPayload.AcceptPartial,
		"first_min_partial_amount": paymentLinkPayload.FirstMinPartialAmount,
		"expire_by":                paymentLinkPayload.ExpireBy,
		"reference_id":             paymentLinkPayload.ReferenceID,
		"description":              paymentLinkPayload.Description,
		"customer": map[string]interface{}{
			"name":   paymentLinkPayload.Customer.Name,
			"email":  paymentLinkPayload.Customer.Email,
			"mobile": paymentLinkPayload.Customer.Mobile,
		},
		"notify": map[string]interface{}{
			"email": paymentLinkPayload.Notify.Email,
		},
		"reminder_enable": paymentLinkPayload.ReminderEnable,
		"notes":           paymentLinkPayload.Notes,
		"callback_url":    paymentLinkPayload.CallbackURL,
		"callback_method": paymentLinkPayload.CallbackMethod,
	}

	return paymentPayload
}

func (g *gateway) toCreatePaymentLinkRequest(req dto.CreatePaymentCommand) createPaymentLinkRequest {
	return createPaymentLinkRequest{
		Amount:                req.Amount,
		Currency:              payments.IndCurrnecy,
		AcceptPartial:         false,
		FirstMinPartialAmount: 0,
		ExpireBy:              0,
		ReferenceID:           req.PaymentID,
		Description:           req.Description,
		Customer: customer{
			Name:   req.Customer.Name,
			Email:  req.Customer.Email,
			Mobile: req.Customer.Mobile,
		},
		Notify: notify{
			Email: req.SendEmail,
		},
		ReminderEnable: req.SendSMS,
		Notes:          req.Metadata,
		CallbackURL:    g.client.PaymentConfig.CallbackUrl,
		CallbackMethod: "POST",
	}
}
func (g *gateway) VerifySignature(payload []byte, signature string) (bool, error) {
	mac := hmac.New(sha256.New, []byte(g.client.PaymentConfig.WebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return false, errors.New("payment verification failed")
	}
	return true, nil
}
