package migration

import (
	"hospital-backend/pkg/db"

	"gorm.io/gorm"
)

// ensureSharedForeignKeys adds FKs for models AutoMigrated in this package
// (no dedicated package migrate helper owns them).
func ensureSharedForeignKeys(gdb *gorm.DB) error {
	return db.EnsureForeignKeys(gdb, []string{
		// Central
		`ALTER TABLE organisations
		   ADD CONSTRAINT fk_organisations_tenant
		   FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE subscriptions
		   ADD CONSTRAINT fk_subscriptions_tenant
		   FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE subscriptions
		   ADD CONSTRAINT fk_subscriptions_plan
		   FOREIGN KEY (plan_id) REFERENCES plans(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE customers
		   ADD CONSTRAINT fk_customers_tenant
		   FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE customer_signup_codes
		   ADD CONSTRAINT fk_customer_signup_codes_user
		   FOREIGN KEY (user_id) REFERENCES customers(id)
		   ON UPDATE CASCADE ON DELETE CASCADE`,

		// Org links / RBAC
		`ALTER TABLE patients
		   ADD CONSTRAINT fk_patients_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE users
		   ADD CONSTRAINT fk_users_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE users
		   ADD CONSTRAINT fk_users_department
		   FOREIGN KEY (department_id) REFERENCES departments(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE users
		   ADD CONSTRAINT fk_users_role
		   FOREIGN KEY (role_id) REFERENCES roles(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE departments
		   ADD CONSTRAINT fk_departments_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE roles
		   ADD CONSTRAINT fk_roles_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE role_permissions
		   ADD CONSTRAINT fk_role_permissions_role
		   FOREIGN KEY (role_id) REFERENCES roles(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE role_permissions
		   ADD CONSTRAINT fk_role_permissions_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,

		// Prescriptions (after appointments + medicines exist)
		`ALTER TABLE prescriptions
		   ADD CONSTRAINT fk_prescriptions_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE prescriptions
		   ADD CONSTRAINT fk_prescriptions_patient
		   FOREIGN KEY (patient_id) REFERENCES patients(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE prescriptions
		   ADD CONSTRAINT fk_prescriptions_appointment
		   FOREIGN KEY (appointment_id) REFERENCES appointments(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE prescriptions
		   ADD CONSTRAINT fk_prescriptions_prescribed_by
		   FOREIGN KEY (prescribed_by) REFERENCES users(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE prescription_items
		   ADD CONSTRAINT fk_prescription_items_prescription
		   FOREIGN KEY (prescription_id) REFERENCES prescriptions(id)
		   ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE prescription_items
		   ADD CONSTRAINT fk_prescription_items_medicine
		   FOREIGN KEY (medicine_id) REFERENCES medicines(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,

		// Organisation schedules
		`ALTER TABLE organisation_schedules
		   ADD CONSTRAINT fk_organisation_schedules_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,

		// Notifications
		`ALTER TABLE notifications
		   ADD CONSTRAINT fk_notifications_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE notifications
		   ADD CONSTRAINT fk_notifications_patient
		   FOREIGN KEY (patient_id) REFERENCES patients(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE notifications
		   ADD CONSTRAINT fk_notifications_employee
		   FOREIGN KEY (employee_id) REFERENCES users(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE notification_attempts
		   ADD CONSTRAINT fk_notification_attempts_notification
		   FOREIGN KEY (notification_id) REFERENCES notifications(id)
		   ON UPDATE CASCADE ON DELETE CASCADE`,

		// Stock movements
		`ALTER TABLE medicine_stock_movements
		   ADD CONSTRAINT fk_medicine_stock_movements_medicine
		   FOREIGN KEY (medicine_id) REFERENCES medicines(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE medicine_stock_movements
		   ADD CONSTRAINT fk_medicine_stock_movements_inventory
		   FOREIGN KEY (medicine_inventory_id) REFERENCES medicine_inventories(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE medicine_stock_movements
		   ADD CONSTRAINT fk_medicine_stock_movements_organisation
		   FOREIGN KEY (organisation_id) REFERENCES organisations(id)
		   ON UPDATE CASCADE ON DELETE RESTRICT`,
	})
}
