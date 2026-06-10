package appointments

import (
	"hospital-backend/internal/admins"
	"testing"
)

func Test_slot(t *testing.T) {
	var scheduleModel admins.OrganisationSchedule
	scheduleModel.OrganisationID = "1"
	scheduleModel.DayOfWeek = []string{"SUN", "SAT"}
	scheduleModel.StartTime = "09:00"
	scheduleModel.EndTime = "17:00"
	scheduleModel.SlotDuration = 30
	scheduleModel.BreakStartTime = "13:00"
	scheduleModel.BreakEndTime = "14:00"
	scheduleModel.IsClosed = false

	appointmentService := AppointmentService{
		OrganisationSchedule: &admins.OrganisationScheduleService{},
	}
	NewAppointmentService(nil, &admins.OrganisationScheduleService{})
	appointments := []Appointment{}
	availableSlots, err := appointmentService.checkIfSlotAvailable(appointments, scheduleModel)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("available slots: %d", len(availableSlots))
}
