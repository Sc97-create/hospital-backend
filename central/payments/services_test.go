package payments_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-backend/central/internalapi"
	"hospital-backend/central/payments"
	dto "hospital-backend/central/payments/dto"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"
)

func TestCreateOrder(t *testing.T) {
	log := servicetest.NopLogger()
	svc := payments.NewPaymentService(internalapi.New("http://example", "id", "secret"))

	_, err := svc.CreateOrder(log, context.Background(), dto.CreateOrderRequest{Amount: 0, Currency: "INR"})
	if !errors.Is(err, wrapError.ErrInvalidAmount) {
		t.Fatalf("err=%v", err)
	}
	_, err = svc.CreateOrder(log, context.Background(), dto.CreateOrderRequest{Amount: 10, Currency: "USD"})
	if !errors.Is(err, wrapError.ErrInvalidCurrency) {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateOrderSuccess(t *testing.T) {
	log := servicetest.NopLogger()
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(dto.CreateOrderResponse{ID: "order_abc", Amount: 10000, Currency: "INR", Status: "created"})
	}))
	defer srv.Close()

	svc := payments.NewPaymentService(internalapi.New(srv.URL, "hb_id", "hb_secret"))
	got, err := svc.CreateOrder(log, context.Background(), dto.CreateOrderRequest{
		Amount: 10000, Currency: "inr", Receipt: "r1",
	})
	if err != nil || got.ID != "order_abc" || gotPath != "/api/v1/hospital/internal/payment/createOrder" {
		t.Fatalf("path=%s got=%+v err=%v", gotPath, got, err)
	}
}
