package razorpay

import (
	"context"
	// "bytes"
	// "encoding/json"
	// "fmt"
	"hospital-backend/internal/payments/dto"
	// "io"
	// "net/http"
)

func NewClient(baseUrl string, apiKey string, apiSecret string) *RazorpayConfig {
	return &RazorpayConfig{
		BaseUrl:   baseUrl,
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
	}
}

func (c *RazorpayConfig) CreatePaymentLink(ctx context.Context, req createPaymentLinkRequest) (dto.CreatePaymentResponse, error) {
	// TODO: remove mock once Razorpay integration is ready for live testing
	_ = ctx
	return dto.CreatePaymentResponse{
		PaymentLinkID: "plink_mock_" + req.ReferenceID,
		PaymentURL:    "https://rzp.io/i/mock-" + req.ReferenceID,
		ReferenceID:   req.ReferenceID,
	}, nil

	// url := fmt.Sprintf("%s/payment-links", c.BaseUrl)
	// body, err := json.Marshal(req)
	// if err != nil {
	// 	return dto.CreatePaymentResponse{}, err
	// }

	// httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	// if err != nil {
	// 	return dto.CreatePaymentResponse{}, err
	// }
	// httpReq.Header.Set("Content-Type", "application/json")
	// httpReq.SetBasicAuth(c.ApiKey, c.ApiSecret)

	// resp, err := http.DefaultClient.Do(httpReq)
	// if err != nil {
	// 	return dto.CreatePaymentResponse{}, err
	// }
	// defer resp.Body.Close()

	// respBody, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return dto.CreatePaymentResponse{}, err
	// }
	// if resp.StatusCode >= http.StatusBadRequest {
	// 	return dto.CreatePaymentResponse{}, fmt.Errorf("razorpay payment link failed: %s", string(respBody))
	// }

	// var razorResp paymentLinkResponse
	// if err := json.Unmarshal(respBody, &razorResp); err != nil {
	// 	return dto.CreatePaymentResponse{}, err
	// }

	// return dto.CreatePaymentResponse{
	// 	PaymentLinkID: razorResp.ID,
	// 	PaymentURL:    razorResp.ShortURL,
	// 	ReferenceID:   razorResp.ReferenceID,
	// }, nil
}
