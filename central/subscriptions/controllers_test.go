package subscriptions_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"hospital-backend/central/plans"
	"hospital-backend/central/subscriptions"
	"hospital-backend/internal/testutil/controllertest"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestControllerGetByTenant(t *testing.T) {
	past := time.Now().Add(-time.Hour)
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
	}}
	svc := subscriptions.NewSubscriptionService(nil, repo, nil, nil, nil, "")
	ctrl := subscriptions.NewISubscriptionController(svc)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/getByTenant/:tenant_id", ctrl.GetByTenant)
	})

	resp, body := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/getByTenant/tenant-1",
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
	var got struct {
		Subscription struct {
			SubscriptionID string `json:"subscription_id"`
			Expired        bool   `json:"expired"`
			Plan           struct {
				Name string `json:"name"`
			} `json:"plan"`
		} `json:"subscription"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Subscription.SubscriptionID != "sub-1" || got.Subscription.Plan.Name != "Clinic" || !got.Subscription.Expired {
		t.Fatalf("body=%s", body)
	}

	missing := subscriptions.NewSubscriptionService(nil, &stubSubRepo{}, nil, nil, nil, "")
	app = controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/getByTenant/:tenant_id", subscriptions.NewISubscriptionController(missing).GetByTenant)
	})
	resp, _ = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/getByTenant/missing",
	})
	controllertest.AssertStatus(t, resp, fiber.StatusNotFound)
}

func TestControllerCreate(t *testing.T) {
	free := plans.Plan{ID: "plan-free-trial", Name: plans.PlanNameFreeTrial, Status: "active"}
	standard := plans.Plan{ID: "plan-standard", Name: plans.PlanNameStandard, MonthlyPrice: 2000, Discount: 10, Status: "active"}
	customerID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	t.Run("missing plan", func(t *testing.T) {
		ctrl := subscriptions.NewISubscriptionController(subscriptions.NewSubscriptionService(nil, &stubSubRepo{}, &stubPlanReader{plan: free}, &stubOrderCreator{}, &stubTenantActivator{}, ""))
		app := controllertest.NewApp(t, func(app *fiber.App) {
			app.Post("/create", ctrl.Create)
		})
		resp, _ := controllertest.Do(t, app, controllertest.Request{Path: "/create", Body: `{"tenant_id":"tenant-1"}`})
		controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
	})

	t.Run("free trial", func(t *testing.T) {
		ctrl := subscriptions.NewISubscriptionController(subscriptions.NewSubscriptionService(nil, &stubSubRepo{}, &stubPlanReader{plan: free}, &stubOrderCreator{}, &stubTenantActivator{}, ""))
		app := controllertest.NewApp(t, func(app *fiber.App) {
			app.Post("/create", controllertest.WithUserID(customerID), ctrl.Create)
		})
		resp, body := controllertest.Do(t, app, controllertest.Request{
			Path: "/create",
			Body: `{"plan_id":"plan-free-trial","tenant_id":"tenant-1","hospital_name":"Acme"}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
		var got struct {
			Status       string  `json:"status"`
			BillingCycle string  `json:"billing_cycle"`
			Price        float64 `json:"price"`
			OrderID      string  `json:"order_id"`
			HospitalName string  `json:"hospital_name"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		if got.Status != "active" || got.BillingCycle != "7d" || got.Price != 0 || got.OrderID != "" || got.HospitalName != "Acme" {
			t.Fatalf("body=%s", body)
		}
	})

	t.Run("paid plan returns checkout", func(t *testing.T) {
		orders := &stubOrderCreator{orderID: "order_xyz"}
		billing := &stubBilling{}
		svc := subscriptions.NewSubscriptionService(subscriptionOrgDB(t), &stubSubRepo{}, &stubPlanReader{plan: standard}, orders, &stubTenantActivator{}, "rzp_test_key")
		svc.Billing = billing
		ctrl := subscriptions.NewISubscriptionController(svc)
		app := controllertest.NewApp(t, func(app *fiber.App) {
			app.Post("/create", controllertest.WithUserID(customerID), ctrl.Create)
		})
		resp, body := controllertest.Do(t, app, controllertest.Request{
			Path: "/create",
			Body: `{"plan_id":"plan-standard","tenant_id":"tenant-1","billing_cycle":12}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
		var got struct {
			Status  string  `json:"status"`
			OrderID string  `json:"order_id"`
			KeyID   string  `json:"key_id"`
			Price   float64 `json:"price"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		if got.Status != "pending_payment" || got.OrderID != "order_xyz" || got.KeyID != "rzp_test_key" || got.Price != 21600 {
			t.Fatalf("body=%s", body)
		}
		if !orders.called || !billing.called {
			t.Fatalf("order=%v invoice=%v", orders.called, billing.called)
		}
	})
}

func TestControllerUpdate(t *testing.T) {
	standard := plans.Plan{ID: "plan-standard", Name: plans.PlanNameStandard, MonthlyPrice: 2000, Discount: 10, Status: "active"}
	existing := subscriptions.Subscription{ID: "sub-1", TenantID: "tenant-1", Status: "active", BillingCycle: "7d"}
	orders := &stubOrderCreator{orderID: "order_renew"}
	svc := subscriptions.NewSubscriptionService(subscriptionOrgDB(t), &stubSubRepo{byID: existing}, &stubPlanReader{plan: standard}, orders, &stubTenantActivator{}, "rzp_test_key")
	svc.Billing = &stubBilling{}
	ctrl := subscriptions.NewISubscriptionController(svc)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/update", controllertest.WithUserID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), ctrl.Update)
	})

	resp, _ := controllertest.Do(t, app, controllertest.Request{Path: "/update", Body: `{"plan_id":"plan-standard"}`})
	controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)

	resp, body := controllertest.Do(t, app, controllertest.Request{
		Path: "/update",
		Body: `{"subscription_id":"sub-1","plan_id":"plan-standard","billing_cycle":6}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
	var got struct {
		Status  string `json:"status"`
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "pending_payment" || got.OrderID != "order_renew" {
		t.Fatalf("body=%s", body)
	}
}

func TestControllerConfirmCheckout(t *testing.T) {
	billing := &stubBilling{subscriptionID: "sub-1"}
	repo := &stubSubRepo{byID: subscriptions.Subscription{ID: "sub-1", Status: "pending_payment"}}
	svc := subscriptions.NewSubscriptionService(nil, repo, &stubPlanReader{}, &stubOrderCreator{}, &stubTenantActivator{}, "rzp_test_key")
	svc.Billing = billing
	ctrl := subscriptions.NewISubscriptionController(svc)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/confirmCheckout", ctrl.ConfirmCheckout)
	})

	resp, _ := controllertest.Do(t, app, controllertest.Request{Path: "/confirmCheckout", Body: `{"razorpay_order_id":"order_1"}`})
	controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)

	resp, body := controllertest.Do(t, app, controllertest.Request{
		Path: "/confirmCheckout",
		Body: `{"razorpay_payment_id":"pay_1","razorpay_order_id":"order_1","razorpay_signature":"sig"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
	var got struct {
		SubscriptionID string `json:"subscription_id"`
		PaymentID      string `json:"payment_id"`
		Status         string `json:"status"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if !billing.confirmCalled || got.SubscriptionID != "sub-1" || got.PaymentID != "pay_1" || got.Status != "pending_payment" {
		t.Fatalf("body=%s", body)
	}
}

func subscriptionOrgDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:suborg?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS organisations (id text primary key, tenant_id text, is_primary integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT OR IGNORE INTO organisations (id, tenant_id, is_primary) VALUES ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'tenant-1', 1)`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestControllerCheckEnd(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:subcheckendctrl?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS organisations (id text primary key, tenant_id text, is_primary integer)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO organisations (id, tenant_id) VALUES ('org-1', 'tenant-1')`).Error; err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Minute)
	repo := &stubSubRepo{byOrder: subscriptions.Subscription{
		ID: "sub-1", TenantID: "tenant-1", BillingCycle: "6", EndAt: &past,
	}}
	ctrl := subscriptions.NewISubscriptionController(subscriptions.NewSubscriptionService(db, repo, nil, nil, nil, ""))
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/checkEnd/:organisation_id", ctrl.CheckEnd)
	})

	resp, body := controllertest.Do(t, app, controllertest.Request{Method: http.MethodGet, Path: "/checkEnd/org-1"})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
	var got struct {
		Ended   bool   `json:"ended"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Ended || got.Message != "subscription ended" {
		t.Fatalf("body=%s", body)
	}

	resp, _ = controllertest.Do(t, app, controllertest.Request{Method: http.MethodGet, Path: "/checkEnd/missing"})
	controllertest.AssertStatus(t, resp, fiber.StatusNotFound)
}

func TestControllerBrokenInputsDoNotPanic(t *testing.T) {
	wired := subscriptions.NewSubscriptionService(nil, &stubSubRepo{}, nil, nil, nil, "")
	ctrl := subscriptions.NewISubscriptionController(wired)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/create", ctrl.Create)
		app.Post("/confirmCheckout", ctrl.ConfirmCheckout)
		app.Get("/getByTenant/:tenant_id", subscriptions.NewISubscriptionController(
			subscriptions.NewSubscriptionService(nil, nil, nil, nil, nil, ""),
		).GetByTenant)
		app.Get("/checkEnd/:organisation_id", ctrl.CheckEnd)
	})

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{name: "malformed json", method: http.MethodPost, path: "/create", body: `{`, status: fiber.StatusBadRequest},
		{name: "plan id wrong type", method: http.MethodPost, path: "/create", body: `{"plan_id":1,"tenant_id":"tenant-1"}`, status: fiber.StatusBadRequest},
		{name: "missing plan dependency", method: http.MethodPost, path: "/create", body: `{"plan_id":"plan-1","tenant_id":"tenant-1"}`, status: fiber.StatusInternalServerError},
		{name: "missing billing dependency", method: http.MethodPost, path: "/confirmCheckout", body: `{"razorpay_payment_id":"pay_1","razorpay_order_id":"order_1","razorpay_signature":"sig"}`, status: fiber.StatusInternalServerError},
		{name: "missing repository", method: http.MethodGet, path: "/getByTenant/tenant-1", status: fiber.StatusInternalServerError},
		{name: "missing database", method: http.MethodGet, path: "/checkEnd/org-1", status: fiber.StatusInternalServerError},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := controllertest.Do(t, app, controllertest.Request{Method: tt.method, Path: tt.path, Body: tt.body})
			controllertest.AssertStatus(t, resp, tt.status)
			if len(body) == 0 {
				t.Fatal("expected an error body")
			}
		})
	}
}
