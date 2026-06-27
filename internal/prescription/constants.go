package prescription

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusPending  Status = "pending"
	StatusDraft    Status = "draft"
	StatusSent     Status = "sent"
	Days           string = "days"
	Weeks          string = "weeks"
	Month          string = "months"
)
