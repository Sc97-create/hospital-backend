package migration

import (
	"hospital-backend/database"
	"hospital-backend/internal/admins"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/bedmanagement"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/jwt"
	"hospital-backend/internal/license"
	"hospital-backend/internal/medicine/medmigration"
	"hospital-backend/internal/modules"
	"hospital-backend/internal/notifications"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/prescription"
	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/roles"
	"log"
)

func Migrate() (err error) {
	err = database.PostgreClient.AutoMigrate(&organisation.Organisation{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&license.License{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&employee.User{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&jwt.RefreshToken{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&patient.Patient{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&department.Department{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&roles.Role{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&permissions.Permission{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&rolepermissions.RolePermission{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&prescription.Prescription{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&prescription.PrescriptionItems{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&modules.Modules{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = bedmanagement.Migrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = medmigration.Automigrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = appointments.AutoMigrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&admins.OrganisationSchedule{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&notifications.Notification{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = database.PostgreClient.AutoMigrate(&notifications.NotificationAttempts{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	return

}
