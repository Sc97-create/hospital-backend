package dto

import (
	"time"
)

type CreatePaymentCommand struct {
	InvoiceID   string  `json:"invoice_id"`
	PatientID   string  `json:"patient_id"`
	InitiatedBy string  `json:"initiated_by"`
	Source      string  `json:"source"`
	Channel     string  `json:"channel"`
	Amount      float64 `json:"amount"`
	PaymentID   string  `json:"payment_id"`
	//Currency    string
	Customer    CustomerInfo
	Description string
	ExpiresAt   time.Time
	Metadata    map[string]string
	SendSMS     bool
	SendEmail   bool
	//CallbackURL string
}
type CustomerInfo struct {
	Name   string
	Email  string
	Mobile string
}
type CreatePaymentResponse struct {
	RequestPayload map[string]interface{} `json:"request_payload"`
	PaymentID      string                 `json:"payment_id"`
	PaymentURL     string                 `json:"payment_url"`
}

type GetPaymentRequest struct {
	PaymentID string `json:"payment_id"`
}

type GetPaymentResponse struct {
	PaymentID  string `json:"payment_id"`
	PaymentURL string `json:"payment_url"`
}

type UpdatePaymentRequest struct {
	PaymentID  string `json:"payment_id"`
	PaymentURL string `json:"payment_url"`
}

type UpdatePaymentResponse struct {
	PaymentID  string `json:"payment_id"`
	PaymentURL string `json:"payment_url"`
}
