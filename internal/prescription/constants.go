package prescription

type Status string

const (
	StatusActive            Status = "active"
	StatusInactive          Status = "inactive"
	StatusPending           Status = "pending"
	StatusDraft             Status = "draft"
	StatusSent              Status = "sent"
	StatusFullyDispensed    Status = "fully_dispensed"    // all prescription items dispensed
	StatusPartiallyDispensed Status = "partially_dispensed" // some items still pending
	Days                    string = "days"
	Weeks                   string = "weeks"
	Month                   string = "months"
)
