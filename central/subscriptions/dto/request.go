package dto

import "time"

// CreateSubscriptionPayload creates an active subscription for an existing tenant.
// billing_cycle (months: 4, 6, or 12) is required for paid plans; ignored for free trial.
type CreateSubscriptionPayload struct {
	PlanID       string  `json:"plan_id"`
	TenantID     string  `json:"tenant_id"`
	HospitalName string  `json:"hospital_name"`
	Price        float64 `json:"price"`
	BillingCycle int     `json:"billing_cycle"`
}

// CreateSubscriptionResult is returned after a successful subscription create.
type CreateSubscriptionResult struct {
	SubscriptionID string     `json:"subscription_id"`
	TenantID       string     `json:"tenant_id"`
	HospitalName   string     `json:"hospital_name"`
	Price          float64    `json:"price"`
	BillingCycle   string     `json:"billing_cycle"`
	Status         string     `json:"status"`
	OrderID        string     `json:"order_id,omitempty"`
	KeyID          string     `json:"key_id,omitempty"`
	StartAt        *time.Time `json:"start_at,omitempty"`
	EndAt          *time.Time `json:"end_at,omitempty"`
}

// ConfirmCheckoutPayload is sent by the frontend after Razorpay Checkout success.
type ConfirmCheckoutPayload struct {
	RazorpayPaymentID string `json:"razorpay_payment_id"`
	RazorpayOrderID   string `json:"razorpay_order_id"`
	RazorpaySignature string `json:"razorpay_signature"`
}

// ConfirmCheckoutResult is returned after checkout payment is verified and saved.
type ConfirmCheckoutResult struct {
	SubscriptionID string `json:"subscription_id"`
	OrderID        string `json:"order_id"`
	PaymentID      string `json:"payment_id"`
	Status         string `json:"status"`
}

// CheckEndResult reports whether the organisation's trial or paid subscription has reached end_at.
type CheckEndResult struct {
	OrganisationID string     `json:"organisation_id"`
	Message        string     `json:"message"`
	Ended          bool       `json:"ended"`
	EndAt          *time.Time `json:"end_at,omitempty"`
}
