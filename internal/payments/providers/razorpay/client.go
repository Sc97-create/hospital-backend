package razorpay

import (
	"context"
	"fmt"

	"hospital-backend/internal/payments/dto"

	rzp "github.com/razorpay/razorpay-go"
)

func NewClient(baseUrl string, apiKey string, apiSecret string) *RazorpayConfig {
	return &RazorpayConfig{
		BaseUrl:   baseUrl, // retained for config compatibility; SDK uses the default Razorpay API host
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
		Client:    rzp.NewClient(apiKey, apiSecret),
	}
}

func (c *RazorpayConfig) CreatePaymentLink(_ context.Context, req createPaymentLinkRequest) (dto.CreatePaymentResponse, error) {
	if c.Client == nil {
		return dto.CreatePaymentResponse{}, fmt.Errorf("razorpay client is not initialized")
	}

	body, err := c.Client.PaymentLink.Create(toPaymentLinkData(req), nil)
	if err != nil {
		return dto.CreatePaymentResponse{}, fmt.Errorf("razorpay payment link failed: %w", err)
	}

	return dto.CreatePaymentResponse{
		PaymentLinkID: asString(body["id"]),
		PaymentURL:    asString(body["short_url"]),
		ReferenceID:   asString(body["reference_id"]),
	}, nil
}

// CreateOrder creates a Razorpay order via the official SDK (server-side Orders API).
func (c *RazorpayConfig) CreateOrder(_ context.Context, amountPaise int64, currency, receipt string, notes map[string]string) (map[string]interface{}, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("razorpay client is not initialized")
	}
	data := map[string]interface{}{
		"amount":   amountPaise,
		"currency": currency,
	}
	if receipt != "" {
		data["receipt"] = receipt
	}
	if len(notes) > 0 {
		noteMap := make(map[string]interface{}, len(notes))
		for k, v := range notes {
			noteMap[k] = v
		}
		data["notes"] = noteMap
	}
	body, err := c.Client.Order.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay order create failed: %w", err)
	}
	return body, nil
}

func toPaymentLinkData(req createPaymentLinkRequest) map[string]interface{} {
	data := map[string]interface{}{
		"amount":   req.Amount,
		"currency": req.Currency,
		"accept_partial": req.AcceptPartial,
		"reference_id":   req.ReferenceID,
		"description":    req.Description,
		"customer": map[string]interface{}{
			"name":    req.Customer.Name,
			"email":   req.Customer.Email,
			"contact": req.Customer.Contact,
		},
		"notify": map[string]interface{}{
			"email": req.Notify.Email,
		},
		"reminder_enable": req.ReminderEnable,
	}
	if req.FirstMinPartialAmount > 0 {
		data["first_min_partial_amount"] = req.FirstMinPartialAmount
	}
	if req.ExpireBy > 0 {
		data["expire_by"] = req.ExpireBy
	}
	if len(req.Notes) > 0 {
		notes := make(map[string]interface{}, len(req.Notes))
		for k, v := range req.Notes {
			notes[k] = v
		}
		data["notes"] = notes
	}
	if req.CallbackURL != "" {
		data["callback_url"] = req.CallbackURL
		data["callback_method"] = req.CallbackMethod
	}
	if req.UPILink {
		data["upi_link"] = true
	}
	return data
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}
