package error

import "errors"

// Auth / session client-facing and internal classification errors.
var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLoginFailed        = errors.New("login failed")
	ErrSessionExpired     = errors.New("session expired")
	ErrRefreshFailed      = errors.New("refresh failed")

	// ErrRefreshSession covers missing/invalid/expired refresh tokens (map to HTTP 401).
	// ErrRefreshInternal covers rotate/DB failures after a valid session (map to HTTP 500).
	ErrRefreshSession  = errors.New("refresh session invalid")
	ErrRefreshInternal = errors.New("refresh internal error")

	// Patient domain
	ErrOrganisationNotFound = errors.New("organisation not found")
	ErrPatientNotFound      = errors.New("patient not found")
	ErrPatientAlreadyExists = errors.New("patient already exists")
	ErrPatientCreateFailed  = errors.New("failed to create patient")
	ErrPatientFetchFailed   = errors.New("failed to fetch patient")
	ErrPatientsFetchFailed  = errors.New("failed to fetch patients")

	// Billing / invoice domain
	ErrInvoiceNotFound      = errors.New("invoice not found")
	ErrInvoiceAlreadyExists = errors.New("invoice already exists for this prescription")
)
