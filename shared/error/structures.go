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
	ErrOrganisationNotFound     = errors.New("organisation not found")
	ErrOrganisationCreateFailed = errors.New("failed to create organisation")
	ErrOrganisationUpdateFailed = errors.New("failed to update organisation")
	ErrOrganisationFetchFailed  = errors.New("failed to fetch organisation")
	ErrPatientNotFound          = errors.New("patient not found")
	ErrPatientAlreadyExists     = errors.New("patient already exists")
	ErrPatientCreateFailed      = errors.New("failed to create patient")
	ErrPatientFetchFailed       = errors.New("failed to fetch patient")
	ErrPatientsFetchFailed      = errors.New("failed to fetch patients")
	ErrEmployeesFetchFailed     = errors.New("failed to fetch employees")

	// Appointment domain
	ErrOrgScheduleNotFound     = errors.New("organisation schedule not found")
	ErrOrgScheduleCreateFailed = errors.New("failed to create organisation schedule")
	ErrOrgScheduleFetchFailed  = errors.New("failed to fetch organisation schedule")
	ErrAppointmentNotFound     = errors.New("appointment not found")
	ErrAppointmentCreateFailed = errors.New("failed to create appointment")
	ErrAppointmentFetchFailed  = errors.New("failed to fetch appointment")
	ErrAppointmentsFetchFailed = errors.New("failed to fetch appointments")
	ErrAppointmentUpdateFailed = errors.New("failed to update appointment")
	ErrAppointmentSlotsFailed  = errors.New("failed to fetch slots")

	// Prescription domain
	ErrPrescriptionNotFound         = errors.New("prescription not found")
	ErrPrescriptionItemNotFound     = errors.New("prescription item not found")
	ErrMedicineAlreadyPresent       = errors.New("medicine already present in prescription")
	ErrCannotEditDispensedItem      = errors.New("cannot edit dispensed item")
	ErrPrescriptionCreateFailed     = errors.New("failed to create prescription")
	ErrPrescriptionFetchFailed      = errors.New("failed to fetch prescription")
	ErrPrescriptionsFetchFailed     = errors.New("failed to fetch prescriptions")
	ErrPrescriptionUpdateFailed     = errors.New("failed to update prescription")
	ErrPrescriptionItemUpdateFailed = errors.New("failed to update prescription item")
	ErrMedicineInfoFetchFailed      = errors.New("failed to fetch medicine info")

	// Billing / invoice domain
	ErrInvoiceNotFound           = errors.New("invoice not found")
	ErrInvoiceAlreadyExists      = errors.New("invoice already exists for this prescription")
	ErrInvoiceNotUnpaid          = errors.New("invoice is not unpaid")
	ErrUnsupportedPaymentMode    = errors.New("unsupported payment mode")
	ErrMedicineNotInPrescription = errors.New("medicine not found in prescription")
	ErrQtyExceedsRemaining       = errors.New("dispensed qty exceeds remaining prescribed qty")
	ErrInsufficientStock         = errors.New("insufficient stock")
	ErrInvoiceCreateFailed       = errors.New("failed to create invoice")
	ErrInvoiceFetchFailed        = errors.New("failed to fetch invoice")
	ErrInvoiceUpdateFailed       = errors.New("failed to update invoice")
	ErrPaymentLinkRetryFailed    = errors.New("failed to retry payment link")

	// payment_type discriminator (consultation vs prescription invoices)
	ErrInvalidPaymentType       = errors.New("invalid or unsupported payment type")
	ErrAppointmentAlreadyBilled = errors.New("appointment already has a consultation invoice")
	ErrAppointmentMismatch      = errors.New("appointment does not belong to this patient/organisation")

	// Payments domain
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrPaymentCreateFailed  = errors.New("failed to create payment")
	ErrPaymentConfirmFailed = errors.New("payment confirm failed")
	ErrPaymentFulfillFailed = errors.New("payment fulfillment failed")
	ErrWebhookProcessFailed = errors.New("failed to process webhook")

	// Medicine / supplier domain
	ErrMedicineNotFound       = errors.New("medicine not found")
	ErrMedicinePurchaseFailed = errors.New("failed to create medicine purchase")
	ErrMedicineSearchFailed   = errors.New("failed to search medicine")
	ErrSupplierNotFound       = errors.New("supplier not found")
	ErrSupplierCreateFailed   = errors.New("failed to create supplier")
	ErrSupplierFetchFailed    = errors.New("failed to fetch supplier")

	ErrPasswordUpdateFailed = errors.New("failed to update password")
	ErrPasswordResetFailed  = errors.New("failed to send password reset link")
	ErrPasswordResetTooSoon = errors.New("password reset already requested recently")

	ErrRolePermissionsFetchFailed = errors.New("failed to fetch role permissions")
)
