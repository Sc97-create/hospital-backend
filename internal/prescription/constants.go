package prescription

type Status string

const (
	StatusActive             Status = "active"
	StatusInactive           Status = "inactive"
	StatusPending            Status = "pending"
	StatusDraft              Status = "draft"
	StatusSent               Status = "sent"
	StatusFullyDispensed     Status = "fully_dispensed"     // all prescription items dispensed
	StatusPartiallyDispensed Status = "partially_dispensed" // some items still pending
	StatusPaymentLinkCreated Status = "payment_link_created"
	Days                     string = "Days"
	Weeks                    string = "Weeks"
	Month                    string = "Months"
)
