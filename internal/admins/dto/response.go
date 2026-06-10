package dto

import "time"

type GetResponse struct {
	ID             string    `json:"id"`
	Starttime      time.Time `json:"start_time"`
	Endtime        time.Time `json:"end_time"`
	BreakStarttime time.Time `json:"break_start_time"`
	BreakEndtime   time.Time `json:"break_end_time"`
	Slotduration   int       `json:"slot_duration"`
}
type FormatResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Code    string `json:"code"`
}
