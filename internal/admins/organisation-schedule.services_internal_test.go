package admins

import (
	"testing"

	"hospital-backend/internal/admins/dto"
)

func TestToOrgSchedModel(t *testing.T) {
	svc := NewOrganisationScheduleService(nil)
	req := dto.OrgScheduleReq{
		OrganisationID: "org-1",
		WeekDays:       []string{"SUN"},
		StartTime:      "09:00",
		EndTime:        "17:00",
		SlotDuration:   30,
		BreakStartTime: "13:00",
		BreakEndTime:   "14:00",
		IsClosed:       true,
	}

	model := svc.toOrgSchedModel(req)
	if model.ID == "" {
		t.Fatal("expected generated id")
	}
	if model.OrganisationID != "org-1" || model.StartTime != "09:00" {
		t.Fatalf("unexpected mapping: %+v", model)
	}
	if model.SlotDuration != 30 || !model.IsClosed {
		t.Fatalf("unexpected slot/closed: duration=%d closed=%v", model.SlotDuration, model.IsClosed)
	}
	if model.CreatedAt.IsZero() {
		t.Fatal("expected created_at to be set")
	}
}

func TestToResponseModel(t *testing.T) {
	svc := NewOrganisationScheduleService(nil)
	resp := svc.toResponseModel(OrganisationSchedule{
		ID:             "sched-1",
		StartTime:      "09:00",
		EndTime:        "17:00",
		BreakStartTime: "13:00",
		BreakEndTime:   "14:00",
		SlotDuration:   30,
	})

	if resp.ID != "sched-1" || resp.Slotduration != 30 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Starttime.Hour() != 9 || resp.Endtime.Hour() != 17 {
		t.Fatalf("unexpected times: start=%v end=%v", resp.Starttime, resp.Endtime)
	}
	if resp.BreakStarttime.Hour() != 13 || resp.BreakEndtime.Hour() != 14 {
		t.Fatalf("unexpected break times: start=%v end=%v", resp.BreakStarttime, resp.BreakEndtime)
	}
}
