package prescription

import (
	"testing"

	"hospital-backend/pkg/constants"
)

func TestParseDurationtype(t *testing.T) {
	svc := &PrescriptionItemServ{}

	tests := []struct {
		in   string
		want string
	}{
		{in: constants.Days, want: constants.Days},
		{in: constants.Weeks, want: constants.Weeks},
		{in: constants.Month, want: constants.Month},
		{in: "unknown", want: constants.Days},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := svc.parseDurationtype(tt.in); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCalculateQuantity(t *testing.T) {
	svc := &PrescriptionItemServ{}
	freq := Freq{Morning: 1, Afternoon: 1, Night: 1}

	tests := []struct {
		name         string
		durationDay  int
		durationType string
		want         int
	}{
		{name: "days", durationDay: 7, durationType: constants.Days, want: 21},
		{name: "weeks", durationDay: 2, durationType: constants.Weeks, want: 42},
		{name: "months", durationDay: 1, durationType: constants.Month, want: 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.calculateQuantity(freq, tt.durationDay, tt.durationType)
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestPrescriptionItemParsePagination(t *testing.T) {
	svc := &PrescriptionItemServ{}

	limit, skip := svc.parsePagination(10, 0)
	if limit != 10 || skip != 0 {
		t.Fatalf("page 0: got limit=%d skip=%d", limit, skip)
	}

	limit, skip = svc.parsePagination(10, 3)
	if limit != 10 || skip != 20 {
		t.Fatalf("page 3: got limit=%d skip=%d", limit, skip)
	}
}
