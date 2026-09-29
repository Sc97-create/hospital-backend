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
	createErr  error
	created    *subscriptions.Subscription
	byOrder    subscriptions.Subscription
	byOrderErr error
	updates    map[string]interface{}
	updateID   string
}

func (s *stubSubRepo) Create(_ *zap.Logger, _ *gorm.DB, sub subscriptions.Subscription) error {
	if s.createErr != nil {
		return s.createErr
	}
	cp := sub
	s.created = &cp
	return nil
}

func (s *stubSubRepo) GetByOrderID(_ *zap.Logger, _ string) (subscriptions.Subscription, error) {
	if s.byOrderErr != nil {
		return subscriptions.Subscription{}, s.byOrderErr
	}
	return s.byOrder, nil
}

func (s *stubSubRepo) GetLatestByTenantID(_ *zap.Logger, _ string) (subscriptions.Subscription, error) {
	if s.byOrderErr != nil {
		return subscriptions.Subscription{}, s.byOrderErr
	}
	return s.byOrder, nil
}

func (s *stubSubRepo) UpdateByID(_ *zap.Logger, _ *gorm.DB, id string, updates map[string]interface{}) error {
	s.updateID = id
	s.updates = updates
	if s.created != nil {
		if orderID, ok := updates["order_id"].(string); ok {
			s.created.OrderID = orderID
		}
		if status, ok := updates["status"].(string); ok {
			s.created.Status = status
		}
	}
	return nil
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
			payload: dto.CreateSubscriptionPayload{PlanID: "p1", TenantID: "t1", BillingCycle: 3},
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
				PlanID:       "plan-standard",
				TenantID:     "tenant-1",
				HospitalName: "Acme Hospital",
				BillingCycle: 12,
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
				PlanID:       "plan-standard",
				TenantID:     "tenant-1",
				BillingCycle: 4,
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
			svc := subscriptions.NewSubscriptionService(nil, repo, plan, orders, &stubTenantActivator{}, "rzp_test_key")
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
				if repo.updates["order_id"] != tt.wantOrderID {
					t.Fatalf("persisted order_id=%v", repo.updates["order_id"])
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

func TestActivateByOrderID(t *testing.T) {
	log := servicetest.NopLogger()
	repo := &stubSubRepo{byOrder: subscriptions.Subscription{
		ID:           "sub-1",
		TenantID:     "tenant-1",
		Status:       "pending_payment",
		BillingCycle: "6",
		OrderID:      "order_1",
	}}
	tenant := &stubTenantActivator{}
	svc := subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{}, &stubOrderCreator{}, tenant, "")
	if err := svc.ActivateByOrderID(log, "order_1", "pay_1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.updates["status"] != "active" {
		t.Fatalf("updates=%v", repo.updates)
	}
	if repo.updates["provider_payment_id"] != "pay_1" {
		t.Fatalf("provider_payment_id missing: %v", repo.updates)
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

	repo := &stubSubRepo{byOrder: subscriptions.Subscription{
		ID:           "sub-1",
		TenantID:     "tenant-1",
		Status:       "pending_payment",
		BillingCycle: "6",
		OrderID:      orderID,
	}}
	tenant := &stubTenantActivator{}
	svc := subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{}, &stubOrderCreator{}, tenant, "rzp_test_key")

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
	if repo.updates["provider_payment_id"] != paymentID {
		t.Fatalf("provider_payment_id=%v", repo.updates["provider_payment_id"])
	}
	if repo.updates["payment_signature"] != signature {
		t.Fatalf("payment_signature=%v", repo.updates["payment_signature"])
	}
	if repo.updates["status"] != nil {
		t.Fatalf("status should not change on confirm: %v", repo.updates["status"])
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
