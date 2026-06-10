package dto

type OrgScheduleReq struct {
	OrganisationID string   `json:"organisation_id"`
	WeekDays       []string `json:"week_days"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	SlotDuration   float64  `json:"slot_duration"`
	BreakStartTime string   `json:"break_start_time"`
	BreakEndTime   string   `json:"break_end_time"`
	IsClosed       bool     `json:"is_closed"`
}
