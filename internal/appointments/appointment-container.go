package appointments

import (
	"hospital-backend/internal/admins"

	"gorm.io/gorm"
)

type AppntmentContainer struct {
	Appointmentservice *AppointmentService
}

func AppointmentContainers(db *gorm.DB, orgschedule admins.OrganisationScheduleService) *AppntmentContainer {
	appointmentrepo := NewCommonDB(db)
	appointmentSrv := NewAppointmentService(db, appointmentrepo, &orgschedule)
	return &AppntmentContainer{
		Appointmentservice: appointmentSrv,
	}
}
