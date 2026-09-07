package appointments

import (
	"testing"
	"time"
)

func TestValidateAppointmentFields(t *testing.T) {
	svc := &AppointmentService{}
	future := time.Now().AddDate(0, 0, 1).Format(time.DateOnly)
	start := future + "T09:00:00Z"
	end := future + "T09:30:00Z"

	tests := []struct {
		name    string
		patient string
		doctor  string
		start   string
		end     string
		date    string
		wantErr bool
	}{
		{name: "missing patient", patient: "", doctor: "doc-1", start: start, end: end, date: future, wantErr: true},
		{name: "missing doctor", patient: "pat-1", doctor: "", start: start, end: end, date: future, wantErr: true},
		{name: "invalid start", patient: "pat-1", doctor: "doc-1", start: "bad", end: end, date: future, wantErr: true},
		{name: "past date", patient: "pat-1", doctor: "doc-1", start: start, end: end, date: "2020-01-01", wantErr: true},
		{name: "success", patient: "pat-1", doctor: "doc-1", start: start, end: end, date: future, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateAppointmentFields(tt.start, tt.end, tt.date, tt.patient, tt.doctor, 30)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParsepagination(t *testing.T) {
	svc := &AppointmentService{}

	limit, offset := svc.parsepagination(0, 0)
	if limit != 0 || offset != 0 {
		t.Fatalf("zero inputs: got limit=%d offset=%d", limit, offset)
	}

	limit, offset = svc.parsepagination(20, 2)
	if limit != 20 || offset != 20 {
		t.Fatalf("page 2: got limit=%d offset=%d", limit, offset)
	}
}

func TestTimesOverlap(t *testing.T) {
	svc := &AppointmentService{}
	aStart := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	aEnd := time.Date(2026, 8, 22, 9, 30, 0, 0, time.UTC)
	bStart := time.Date(2026, 8, 22, 9, 15, 0, 0, time.UTC)
	bEnd := time.Date(2026, 8, 22, 9, 45, 0, 0, time.UTC)

	if !svc.timesOverlap(aStart, aEnd, bStart, bEnd) {
		t.Fatal("expected overlap")
	}
	if svc.timesOverlap(aStart, aEnd, aEnd, bEnd) {
		t.Fatal("expected no overlap for adjacent slots")
	}
}

func TestGenerateAppointmentCode(t *testing.T) {
	svc := &AppointmentService{}
	code := svc.generateAppointmentCode()
	if code == "" {
		t.Fatal("expected non-empty code")
	}
}

func TestNormalizeTimeOfDay(t *testing.T) {
	svc := &AppointmentService{}
	in := time.Date(2026, 8, 22, 14, 30, 15, 0, time.UTC)
	got := svc.normalizeTimeOfDay(in)
	if got.Hour() != 14 || got.Minute() != 30 {
		t.Fatalf("unexpected normalized time: %v", got)
	}
}
