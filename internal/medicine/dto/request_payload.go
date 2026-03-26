package dto

import "time"

type RequestPayload struct {
	UserID      string
	Name        string
	Form        string
	Strength    string
	BatchNumber string
	ExpiryDate  time.Time
	Quantity    int64
}
