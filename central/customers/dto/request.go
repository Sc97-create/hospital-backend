package dto

// SignupPayload registers a new customer account.
type SignupPayload struct {
	FullName  string `json:"full_name"`
	WorkEmail string `json:"work_email"`
	Password  string `json:"password"`
}

// SignupResult is returned after a successful customer signup.
type SignupResult struct {
	CustomerID  string `json:"customer_id"`
	WorkEmail   string `json:"work_email"`
	Status      string `json:"status"`
	AccessToken string `json:"access_token"`
}

// VerifyEmailPayload verifies the 6-digit signup code for the authenticated customer.
type VerifyEmailPayload struct {
	Code string `json:"code"`
}

// LoginPayload authenticates a customer by work email and password.
type LoginPayload struct {
	WorkEmail string `json:"work_email"`
	Password  string `json:"password"`
}

// LoginResult is returned after a successful customer login.
type LoginResult struct {
	CustomerID          string `json:"customer_id"`
	WorkEmail           string `json:"work_email"`
	Status              string `json:"status"`
	TenantID            string `json:"tenant_id,omitempty"`
	AccessToken         string `json:"access_token"`
	SubscriptionExpired bool   `json:"subscription_expired"`
}
