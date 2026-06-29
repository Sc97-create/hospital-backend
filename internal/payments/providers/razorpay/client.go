package razorpay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hospital-backend/internal/payments/dto"
	"net/http"
)

func NewClient(baseUrl string, apiKey string, apiSecret string) *RazorpayConfig {
	return &RazorpayConfig{
		BaseUrl:   baseUrl,
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
	}
}

func (c *RazorpayConfig) CreatePaymentLink(ctx context.Context, req createPaymentLinkRequest) (dto.CreatePaymentResponse, error) {
	url := fmt.Sprintf("%s/payment-links", c.BaseUrl)
	body, err := json.Marshal(req)
	if err != nil {
		return dto.CreatePaymentResponse{}, err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return dto.CreatePaymentResponse{}, err
	}
	defer resp.Body.Close()
	var response dto.CreatePaymentResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return dto.CreatePaymentResponse{}, err
	}
	return response, nil
}
