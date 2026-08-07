package appointments

import (
	"hospital-backend/internal/admins"
	"hospital-backend/internal/notifications/service"

	"gorm.io/gorm"
)

type AppntmentContainer struct {
	Db                   *gorm.DB
	Appointmentservice   *AppointmentService
	OrganisationSchedule *admins.OrganisationScheduleService
	NotificationService  *service.Notificationservice
}

func AppointmentContainers(db *gorm.DB, orgschedule *admins.OrganisationScheduleService, notificationServ *service.Notificationservice) *AppntmentContainer {
	appointmentrepo := NewCommonDB(db)
	appointmentSrv := NewAppointmentService(db, appointmentrepo, orgschedule, notificationServ)
	return &AppntmentContainer{
		Db:                   db,
		Appointmentservice:   appointmentSrv,
		OrganisationSchedule: orgschedule,
		NotificationService:  notificationServ,
	}
}
