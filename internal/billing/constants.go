package billing

const (
	StatusUnpaid string = "unpaid"
	StatusPaid   string = "paid"
	Source       string = "link"
	InvPrefix    string = "INV"
)

// PaymentType discriminates what an invoice was raised for. Kept as a plain
// varchar (not a DB enum) so new values (advance, package, refund, ...) are a
// data-only change, not a migration.
type PaymentType string

const (
	PaymentTypeConsultation PaymentType = "consultation"
	PaymentTypePrescription PaymentType = "prescription"
)
