package migration

import (
	"hospital-backend/central/customers"
	"hospital-backend/central/organisations"
	"hospital-backend/central/plans"
	"hospital-backend/central/subscriptions"
	"hospital-backend/central/tenants"
	"hospital-backend/database"
	"hospital-backend/internal/admins"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/bedmanagement"
	"hospital-backend/internal/billing"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/jwt"
	labmigrate "hospital-backend/internal/labService/migrate"
	"hospital-backend/internal/medicine/medmigration"
	"hospital-backend/internal/modules"
	"hospital-backend/internal/notifications"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/payments"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/prescription"
	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/roles"
	"hospital-backend/pkg/types"
	"log"
)

func Migrate() (err error) {
	// Parents first: tenants → plans → subscriptions → customers → orgs.
	err = database.PostgreClient.AutoMigrate(&tenants.Tenant{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = plans.AutoMigrate(database.PostgreClient.GormDriver)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = subscriptions.AutoMigrate(database.PostgreClient.GormDriver)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&customers.Customer{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&customers.SignupCode{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&organisations.Organisation{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	// Departments/roles before users so user FKs can be applied later.
	err = database.PostgreClient.AutoMigrate(&department.Department{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&roles.Role{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&permissions.Permission{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&modules.Modules{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&rolepermissions.RolePermission{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = rolepermissions.RelaxNullableColumns(database.PostgreClient.GormDriver)
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
		return
	}

	// Schedules before appointments (appointments.schedule_id FK).
	err = database.PostgreClient.AutoMigrate(&admins.OrganisationSchedule{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = appointments.AutoMigrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	err = database.PostgreClient.AutoMigrate(&prescription.Prescription{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&prescription.PrescriptionItems{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	err = bedmanagement.Migrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = medmigration.Automigrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&types.MedicineStockMovements{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	err = database.PostgreClient.AutoMigrate(&notifications.Notification{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = notifications.RelaxRecipientConstraints(database.PostgreClient.GormDriver)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = database.PostgreClient.AutoMigrate(&notifications.NotificationAttempts{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	// Billing before payments (payments.invoice_id FK).
	err = billing.Migrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = payments.Migrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	err = labmigrate.Migrate(database.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}

	err = ensureSharedForeignKeys(database.PostgreClient.GormDriver)
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	return
}
