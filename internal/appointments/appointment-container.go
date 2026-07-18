package appointments

import (
	"hospital-backend/internal/admins"
	"hospital-backend/internal/notifications/service"

	"gorm.io/gorm"
)

type AppntmentContainer struct {
	Appointmentservice *AppointmentService
}

func AppointmentContainers(db *gorm.DB, orgschedule admins.OrganisationScheduleService, notificationServ *service.Notificationservice) *AppntmentContainer {
	appointmentrepo := NewCommonDB(db)
	appointmentSrv := NewAppointmentService(db, appointmentrepo, &orgschedule, notificationServ)
	return &AppntmentContainer{
		Appointmentservice: appointmentSrv,
	}
}
