package admins

import (
	"time"

	"github.com/lib/pq"
)

const (
	Monday          = "MON"
	Tuesday         = "TUE"
	Wednesday       = "WED"
	Thursday        = "THU"
	Friday          = "FRI"
	Saturday        = "SAT"
	Sunday          = "SUN"
	StatusOk        = "200"
	OrgSchedCreated = "organisation schedule created successfully"
)

type OrganisationSchedule struct {
	ID             string         `gorm:"primaryKey;type:uuid;not null" json:"id"`
	OrganisationID string         `json:"organisation_id"`
	DayOfWeek      pq.StringArray `json:"day_of_week" gorm:"type:text[]"`
	StartTime      string         `json:"start_time"`
	EndTime        string         `json:"end_time"`
	SlotDuration   int            `json:"slot_duration"` // in minutes
	BreakStartTime string         `json:"break_start_time"`
	BreakEndTime   string         `json:"break_end_time"`
	IsClosed       bool           `json:"is_closed"`
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime;type:timestamp"`
}
