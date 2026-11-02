package subscriptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentdto "hospital-backend/central/payments/dto"
	"hospital-backend/central/plans"
	"hospital-backend/central/subscriptions"
	dto "hospital-backend/central/subscriptions/dto"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubPlanReader struct {
	plan plans.Plan
	err  error
}

func (s *stubPlanReader) GetByID(_ *zap.Logger, _ string) (plans.Plan, error) {
	if s.err != nil {
		return plans.Plan{}, s.err
	}
	return s.plan, nil
}

type stubSubRepo struct {
	createErr   error
	created     *subscriptions.Subscription
	byOrder     subscriptions.Subscription
	byOrderErr  error
	updates     map[string]interface{}
	updateID    string
	withPlan    subscriptions.SubscriptionPlanRow
	withPlanErr error
	byID        subscriptions.Subscription
	byIDErr     error
}

func (s *stubSubRepo) Create(_ *zap.Logger, _ *gorm.DB, sub subscriptions.Subscription) error {
	if s.createErr != nil {
		return s.createErr
	}
	cp := sub
	s.created = &cp
	return nil
}

func (s *stubSubRepo) GetByID(_ *zap.Logger, _ string) (subscriptions.Subscription, error) {
	if s.byIDErr != nil {
		return subscriptions.Subscription{}, s.byIDErr
	}
	if s.byID.ID == "" {
		return subscriptions.Subscription{}, subscriptions.ErrSubscriptionNotFound
	}
	return s.byID, nil
}

func (s *stubSubRepo) GetLatestByTenantID(_ *zap.Logger, _ string) (subscriptions.Subscription, error) {
	if s.byOrderErr != nil {
		return subscriptions.Subscription{}, s.byOrderErr
	}
	return s.byOrder, nil
}

func (s *stubSubRepo) GetLatestWithPlanByTenantID(_ *zap.Logger, _ string) (subscriptions.SubscriptionPlanRow, error) {
	if s.withPlanErr != nil {
		return subscriptions.SubscriptionPlanRow{}, s.withPlanErr
	}
	if s.withPlan.ID == "" {
		return subscriptions.SubscriptionPlanRow{}, subscriptions.ErrSubscriptionNotFound
	}
	return s.withPlan, nil
}

func (s *stubSubRepo) UpdateByID(_ *zap.Logger, _ *gorm.DB, id string, updates map[string]interface{}) error {
	s.updateID = id
	s.updates = updates
	if s.created != nil {
		if status, ok := updates["status"].(string); ok {
			s.created.Status = status
		}
	}
	return nil
}

type stubBilling struct {
	called         bool
	renewal        bool
	customerID     string
	orderID        string
	confirmCalled  bool
	subscriptionID string
	confirmErr     error
}

func (s *stubBilling) CreateSubscriptionInvoice(_ *zap.Logger, _ context.Context, req paymentdto.SubscriptionInvoiceRequest) (paymentdto.SubscriptionInvoiceResponse, error) {
	s.called = true
	s.renewal = req.Renewal
	s.customerID = req.CustomerID
	kind := "subscription_new"
	if req.Renewal {
		kind = "subscription_renewal"
	}
	return paymentdto.SubscriptionInvoiceResponse{InvoiceID: "11111111-1111-1111-1111-111111111111", PaymentType: kind, Status: "unpaid"}, nil
}

func (s *stubBilling) RecordSubscriptionPayment(_ *zap.Logger, _ context.Context, req paymentdto.SubscriptionPaymentRequest) error {
	s.orderID = req.OrderID
	return nil
}

func (s *stubBilling) ConfirmSubscriptionPayment(_ *zap.Logger, _ context.Context, req paymentdto.ConfirmSubscriptionPaymentRequest) (paymentdto.ConfirmSubscriptionPaymentResponse, error) {
	s.confirmCalled = true
	s.orderID = req.OrderID
	if s.confirmErr != nil {
		return paymentdto.ConfirmSubscriptionPaymentResponse{}, s.confirmErr
	}
	id := s.subscriptionID
	if id == "" {
		id = "sub-1"
	}
	return paymentdto.ConfirmSubscriptionPaymentResponse{PaymentID: "pay-1", OrderID: req.OrderID, SubscriptionID: id}, nil
}

type stubOrderCreator struct {
	err     error
	lastReq paymentdto.CreateOrderRequest
	orderID string
	called  bool
}

func (s *stubOrderCreator) CreateOrder(_ *zap.Logger, _ context.Context, req paymentdto.CreateOrderRequest) (paymentdto.CreateOrderResponse, error) {
	s.called = true
	s.lastReq = req
	if s.err != nil {
		return paymentdto.CreateOrderResponse{}, s.err
	}
	id := s.orderID
	if id == "" {
		id = "order_test"
	}
	return paymentdto.CreateOrderResponse{ID: id, Amount: req.Amount, Currency: req.Currency, Status: "created", Entity: "order"}, nil
}

type stubTenantActivator struct {
	called   bool
	tenantID string
	err      error
}

func (s *stubTenantActivator) Activate(_ *zap.Logger, tenantID string) error {
	s.called = true
	s.tenantID = tenantID
	return s.err
}

func TestServiceCreateSubscription(t *testing.T) {
	log := servicetest.NopLogger()
	standard := plans.Plan{
		ID:           "plan-standard",
		Name:         plans.PlanNameStandard,
		MonthlyPrice: 2000,
		Discount:     10,
		Status:       "active",
	}
	freeTrial := plans.Plan{
		ID:           "plan-free-trial",
		Name:         plans.PlanNameFreeTrial,
		MonthlyPrice: 0,
		Discount:     0,
		Status:       "active",
	}

	tests := []struct {
		name        string
		payload     dto.CreateSubscriptionPayload
		plan        *stubPlanReader
		repo        *stubSubRepo
		orders      *stubOrderCreator
		wantErr     error
		wantPrice   float64
		wantCycle   string
		wantStatus  string
		wantEndDay  int
		wantOrder   bool
		wantAmount  int64
		wantOrderID string
	}{
		{
			name:    "missing plan_id",
			payload: dto.CreateSubscriptionPayload{TenantID: "t1", BillingCycle: 4},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "invalid billing cycle for paid plan",
			payload: dto.CreateSubscriptionPayload{PlanID: "p1", TenantID: "t1", BillingCycle: 4},
			plan:    &stubPlanReader{plan: standard},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "plan not found",
			payload: dto.CreateSubscriptionPayload{PlanID: "missing", TenantID: "t1", BillingCycle: 6},
			plan:    &stubPlanReader{err: gorm.ErrRecordNotFound},
			repo:    &stubSubRepo{},
			wantErr: wrapError.ErrPlanNotFound,
		},
		{
			name: "free trial active immediately",
			payload: dto.CreateSubscriptionPayload{
				PlanID:   "plan-free-trial",
				TenantID: "tenant-1",
			},
			plan:       &stubPlanReader{plan: freeTrial},
			repo:       &stubSubRepo{},
			orders:     &stubOrderCreator{},
			wantPrice:  0,
			wantCycle:  "7d",
			wantStatus: "active",
			wantEndDay: 7,
			wantOrder:  false,
		},
		{
			name: "paid plan pending_payment until order.paid",
			payload: dto.CreateSubscriptionPayload{
				PlanID:         "plan-standard",
				TenantID:       "tenant-1",
				HospitalName:   "Acme Hospital",
				BillingCycle:   12,
				CustomerID:     "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				OrganisationID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
			},
			plan:        &stubPlanReader{plan: standard},
			repo:        &stubSubRepo{},
			orders:      &stubOrderCreator{orderID: "order_xyz"},
			wantPrice:   2000 * 0.9 * 12,
			wantCycle:   "12",
			wantStatus:  "pending_payment",
			wantOrder:   true,
			wantAmount:  int64(2000 * 0.9 * 12 * 100),
			wantOrderID: "order_xyz",
		},
		{
			name: "order error returned as-is",
			payload: dto.CreateSubscriptionPayload{
				PlanID:         "plan-standard",
				TenantID:       "tenant-1",
				BillingCycle:   6,
				CustomerID:     "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				OrganisationID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
			},
			plan:    &stubPlanReader{plan: standard},
			repo:    &stubSubRepo{},
			orders:  &stubOrderCreator{err: wrapError.ErrOrderCreateFailed},
			wantErr: wrapError.ErrOrderCreateFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := tt.plan
			if plan == nil {
				plan = &stubPlanReader{plan: standard}
			}
			repo := tt.repo
			if repo == nil {
				repo = &stubSubRepo{}
			}
			orders := tt.orders
			if orders == nil {
				orders = &stubOrderCreator{}
			}
			billing := &stubBilling{}
			svc := subscriptions.NewSubscriptionService(nil, repo, plan, orders, &stubTenantActivator{}, "rzp_test_key")
			svc.Billing = billing
			got, err := svc.CreateSubscription(log, context.Background(), tt.payload)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err=%v want=%v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("status=%q want %q", got.Status, tt.wantStatus)
			}
			if got.Price != tt.wantPrice {
				t.Fatalf("price=%v want=%v", got.Price, tt.wantPrice)
			}
			if got.BillingCycle != tt.wantCycle {
				t.Fatalf("billing_cycle=%q want=%q", got.BillingCycle, tt.wantCycle)
			}
			if orders.called != tt.wantOrder {
				t.Fatalf("order called=%v want=%v", orders.called, tt.wantOrder)
			}
			if tt.wantOrder {
				if orders.lastReq.Currency != "INR" || orders.lastReq.Amount != tt.wantAmount {
					t.Fatalf("order req=%+v want amount=%d INR", orders.lastReq, tt.wantAmount)
				}
				if got.OrderID != tt.wantOrderID || got.KeyID != "rzp_test_key" {
					t.Fatalf("order_id=%q key_id=%q", got.OrderID, got.KeyID)
				}
				if !billing.called || billing.renewal || billing.orderID != tt.wantOrderID {
					t.Fatalf("billing called=%v renewal=%v order=%q", billing.called, billing.renewal, billing.orderID)
				}
				if _, stored := repo.updates["order_id"]; stored {
					t.Fatalf("order_id must stay off the subscription: %v", repo.updates)
				}
			}
			if tt.wantStatus == "active" {
				if repo.created == nil || repo.created.StartAt == nil || repo.created.EndAt == nil {
					t.Fatalf("expected start/end dates on free trial")
				}
				wantEnd := repo.created.StartAt.AddDate(0, 0, tt.wantEndDay)
				if diff := repo.created.EndAt.Sub(wantEnd); diff > time.Second || diff < -time.Second {
					t.Fatalf("end_at=%v want=%v", repo.created.EndAt, wantEnd)
				}
			}
		})
	}
}

func TestActivateBySubscriptionID(t *testing.T) {
	log := servicetest.NopLogger()
	repo := &stubSubRepo{byID: subscriptions.Subscription{
		ID:           "sub-1",
		TenantID:     "tenant-1",
		Status:       "pending_payment",
		BillingCycle: "6",
	}}
	tenant := &stubTenantActivator{}
	svc := subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{}, &stubOrderCreator{}, tenant, "")
	if err := svc.ActivateBySubscriptionID(log, "sub-1", "pay_1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.updates["status"] != "active" {
		t.Fatalf("updates=%v", repo.updates)
	}
	if _, stored := repo.updates["provider_payment_id"]; stored {
		t.Fatalf("provider payment id must stay off the subscription: %v", repo.updates)
	}
	if !tenant.called || tenant.tenantID != "tenant-1" {
		t.Fatalf("tenant activate called=%v id=%q", tenant.called, tenant.tenantID)
	}
}

func TestConfirmCheckoutStoresOnly(t *testing.T) {
	log := servicetest.NopLogger()
	const (
		orderID   = "order_9A33XWu170gUtm"
		paymentID = "pay_29QQoUBi66xm2f"
		signature = "9ef4dffbfd84f1318f6739a3ce19f9d85851857ae648f114332d8401e0949a3d"
	)

	repo := &stubSubRepo{byID: subscriptions.Subscription{
		ID:           "sub-1",
		TenantID:     "tenant-1",
		Status:       "pending_payment",
		BillingCycle: "6",
	}}
	tenant := &stubTenantActivator{}
	billing := &stubBilling{subscriptionID: "sub-1"}
	svc := subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{}, &stubOrderCreator{}, tenant, "rzp_test_key")
	svc.Billing = billing

	got, err := svc.ConfirmCheckout(log, dto.ConfirmCheckoutPayload{
		RazorpayOrderID:   orderID,
		RazorpayPaymentID: paymentID,
		RazorpaySignature: signature,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Status != "pending_payment" || got.SubscriptionID != "sub-1" || got.PaymentID != paymentID {
		t.Fatalf("got=%+v", got)
	}
	if !billing.confirmCalled || billing.orderID != orderID {
		t.Fatalf("confirm called=%v order=%q", billing.confirmCalled, billing.orderID)
	}
	if repo.updates != nil {
		t.Fatalf("subscription row must not change on confirm: %v", repo.updates)
	}
	if tenant.called {
		t.Fatal("tenant must not activate on confirm checkout")
	}
}

func TestCheckEndByOrganisation(t *testing.T) {
	log := servicetest.NopLogger()
	db, err := gorm.Open(sqlite.Open("file:subcheckend?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS organisations (id text primary key, tenant_id text)`).Error; err != nil {
		t.Fatalf("create organisations: %v", err)
	}
	if err := db.Exec(`INSERT INTO organisations (id, tenant_id) VALUES ('org-1', 'tenant-1')`).Error; err != nil {
		t.Fatalf("insert organisation: %v", err)
	}

	past := time.Now().Add(-time.Minute)
	repo := &stubSubRepo{byOrder: subscriptions.Subscription{
		ID:           "sub-1",
		TenantID:     "tenant-1",
		BillingCycle: "7d",
		EndAt:        &past,
	}}
	svc := subscriptions.NewSubscriptionService(db, repo, &stubPlanReader{}, &stubOrderCreator{}, &stubTenantActivator{}, "")

	got, err := svc.CheckEndByOrganisation(log, "org-1")
	if err != nil {
		t.Fatalf("trial check: %v", err)
	}
	if !got.Ended || got.Message != "trial period ended" {
		t.Fatalf("trial got=%+v", got)
	}

	future := time.Now().Add(time.Hour)
	repo.byOrder.BillingCycle = "6"
	repo.byOrder.EndAt = &future
	got, err = svc.CheckEndByOrganisation(log, "org-1")
	if err != nil {
		t.Fatalf("active check: %v", err)
	}
	if got.Ended || got.Message != "active" {
		t.Fatalf("active got=%+v", got)
	}

	ended := time.Now().Add(-time.Second)
	repo.byOrder.EndAt = &ended
	got, err = svc.CheckEndByOrganisation(log, "org-1")
	if err != nil {
		t.Fatalf("paid check: %v", err)
	}
	if !got.Ended || got.Message != "subscription ended" {
		t.Fatalf("paid got=%+v", got)
	}
}

func TestExpiredByTenantID(t *testing.T) {
	log := servicetest.NopLogger()
	past := time.Now().Add(-time.Minute)
	future := time.Now().Add(time.Hour)
	svc := subscriptions.NewSubscriptionService(nil, &stubSubRepo{}, &stubPlanReader{}, &stubOrderCreator{}, &stubTenantActivator{}, "")

	expired, err := svc.ExpiredByTenantID(log, "  ")
	if err != nil || expired {
		t.Fatalf("empty tenant expired=%v err=%v", expired, err)
	}

	repo := &stubSubRepo{byOrderErr: subscriptions.ErrSubscriptionNotFound}
	svc = subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{}, &stubOrderCreator{}, &stubTenantActivator{}, "")
	expired, err = svc.ExpiredByTenantID(log, "tenant-1")
	if err != nil || expired {
		t.Fatalf("missing subscription expired=%v err=%v", expired, err)
	}

	repo = &stubSubRepo{byOrder: subscriptions.Subscription{ID: "sub-1", EndAt: &past}}
	svc = subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{}, &stubOrderCreator{}, &stubTenantActivator{}, "")
	expired, err = svc.ExpiredByTenantID(log, "tenant-1")
	if err != nil || !expired {
		t.Fatalf("past end expired=%v err=%v", expired, err)
	}

	repo.byOrder.EndAt = &future
	expired, err = svc.ExpiredByTenantID(log, "tenant-1")
	if err != nil || expired {
		t.Fatalf("future end expired=%v err=%v", expired, err)
	}
}

func TestGetByTenantID(t *testing.T) {
	log := servicetest.NopLogger()
	past := time.Now().Add(-time.Hour)
	svc := subscriptions.NewSubscriptionService(nil, &stubSubRepo{}, nil, nil, nil, "")

	_, err := svc.GetByTenantID(log, " ")
	if !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("empty tenant err=%v", err)
	}

	_, err = svc.GetByTenantID(log, "tenant-1")
	if !errors.Is(err, wrapError.ErrSubscriptionNotFound) {
		t.Fatalf("missing err=%v", err)
	}

	svc = subscriptions.NewSubscriptionService(nil, &stubSubRepo{withPlanErr: errors.New("db down")}, nil, nil, nil, "")
	_, err = svc.GetByTenantID(log, "tenant-1")
	if !errors.Is(err, wrapError.ErrSubscriptionFetchFailed) {
		t.Fatalf("db err=%v", err)
	}

	repo := &stubSubRepo{withPlan: subscriptions.SubscriptionPlanRow{
		ID:           "sub-1",
		TenantID:     "tenant-1",
		PlanID:       "plan-1",
		Status:       "active",
		BillingCycle: "12",
		Price:        1200,
		EndAt:        &past,
		PlanName:     "Clinic",
		PlanStatus:   "active",
		MonthlyPrice: 100,
		Discount:     10,
		PlanDetails:  map[string]any{"seats": float64(5)},
	}}
	svc = subscriptions.NewSubscriptionService(nil, repo, nil, nil, nil, "")
	got, err := svc.GetByTenantID(log, "tenant-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SubscriptionID != "sub-1" || got.Plan.Name != "Clinic" || got.Plan.MonthlyPrice != 100 || !got.Expired {
		t.Fatalf("got=%+v", got)
	}
	if got.Plan.PlanDetails["seats"] != float64(5) {
		t.Fatalf("plan details=%v", got.Plan.PlanDetails)
	}
}

func TestUpdateSubscription(t *testing.T) {
	log := servicetest.NopLogger()
	standard := plans.Plan{
		ID:           "plan-standard",
		Name:         plans.PlanNameStandard,
		MonthlyPrice: 2000,
		Discount:     10,
		Status:       "active",
	}
	freeTrial := plans.Plan{
		ID:   "plan-free-trial",
		Name: plans.PlanNameFreeTrial,
	}
	existing := subscriptions.Subscription{ID: "sub-1", TenantID: "tenant-1", Status: "active", BillingCycle: "7d"}

	t.Run("missing subscription_id", func(t *testing.T) {
		svc := subscriptions.NewSubscriptionService(nil, &stubSubRepo{}, &stubPlanReader{}, &stubOrderCreator{}, &stubTenantActivator{}, "")
		_, err := svc.UpdateSubscription(log, context.Background(), dto.UpdateSubscriptionPayload{PlanID: "plan-standard"})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("subscription not found", func(t *testing.T) {
		svc := subscriptions.NewSubscriptionService(nil, &stubSubRepo{}, &stubPlanReader{plan: standard}, &stubOrderCreator{}, &stubTenantActivator{}, "")
		_, err := svc.UpdateSubscription(log, context.Background(), dto.UpdateSubscriptionPayload{
			SubscriptionID: "missing", PlanID: "plan-standard", BillingCycle: 12,
		})
		if !errors.Is(err, wrapError.ErrSubscriptionNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("free trial sets dates and skips order", func(t *testing.T) {
		repo := &stubSubRepo{byID: existing}
		orders := &stubOrderCreator{}
		tenant := &stubTenantActivator{}
		svc := subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{plan: freeTrial}, orders, tenant, "rzp_test_key")
		got, err := svc.UpdateSubscription(log, context.Background(), dto.UpdateSubscriptionPayload{
			SubscriptionID: "sub-1",
			PlanID:         "plan-free-trial",
		})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if orders.called || got.OrderID != "" || got.Status != "active" || got.BillingCycle != "7d" || got.Price != 0 {
			t.Fatalf("got=%+v orderCalled=%v", got, orders.called)
		}
		if !tenant.called || repo.updates["plan_id"] != "plan-free-trial" {
			t.Fatalf("updates=%v tenant=%v", repo.updates, tenant.called)
		}
		if _, stored := repo.updates["order_id"]; stored {
			t.Fatalf("order_id must stay off the subscription: %v", repo.updates)
		}
		start, _ := repo.updates["start_at"].(time.Time)
		end, _ := repo.updates["end_at"].(time.Time)
		if start.IsZero() || end.Sub(start) < 6*24*time.Hour {
			t.Fatalf("start=%v end=%v", start, end)
		}
	})

	t.Run("standard plan creates order invoice and payment", func(t *testing.T) {
		repo := &stubSubRepo{byID: existing}
		orders := &stubOrderCreator{orderID: "order_renew"}
		billing := &stubBilling{}
		svc := subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{plan: standard}, orders, &stubTenantActivator{}, "rzp_test_key")
		svc.Billing = billing
		got, err := svc.UpdateSubscription(log, context.Background(), dto.UpdateSubscriptionPayload{
			SubscriptionID: "sub-1",
			PlanID:         "plan-standard",
			HospitalName:   "Acme Hospital",
			BillingCycle:   12,
			CustomerID:     "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			OrganisationID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		wantPrice := 2000 * 0.9 * 12
		if got.OrderID != "order_renew" || got.KeyID != "rzp_test_key" || got.Status != "pending_payment" || got.Price != wantPrice {
			t.Fatalf("got=%+v", got)
		}
		if repo.updates["billing_cycle"] != "12" || repo.updates["plan_id"] != "plan-standard" {
			t.Fatalf("updates=%v", repo.updates)
		}
		if _, stored := repo.updates["order_id"]; stored {
			t.Fatalf("order_id must stay off the subscription: %v", repo.updates)
		}
		if !billing.called || !billing.renewal || billing.orderID != "order_renew" {
			t.Fatalf("billing called=%v renewal=%v order=%q", billing.called, billing.renewal, billing.orderID)
		}
		start, _ := repo.updates["start_at"].(time.Time)
		end, _ := repo.updates["end_at"].(time.Time)
		if start.IsZero() || end.Before(start.AddDate(0, 11, 0)) {
			t.Fatalf("start=%v end=%v", start, end)
		}
		if orders.lastReq.Amount != int64(wantPrice*100) || orders.lastReq.Currency != "INR" || orders.lastReq.Notes["subscription_id"] != "sub-1" {
			t.Fatalf("order=%+v", orders.lastReq)
		}
	})

	t.Run("order failure leaves subscription unchanged", func(t *testing.T) {
		repo := &stubSubRepo{byID: existing}
		orders := &stubOrderCreator{err: wrapError.ErrOrderCreateFailed}
		svc := subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{plan: standard}, orders, &stubTenantActivator{}, "")
		_, err := svc.UpdateSubscription(log, context.Background(), dto.UpdateSubscriptionPayload{
			SubscriptionID: "sub-1",
			PlanID:         "plan-standard",
			BillingCycle:   6,
			CustomerID:     "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			OrganisationID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		})
		if !errors.Is(err, wrapError.ErrOrderCreateFailed) {
			t.Fatalf("err=%v", err)
		}
		if repo.updates != nil {
			t.Fatalf("updates=%v", repo.updates)
		}
	})
}
