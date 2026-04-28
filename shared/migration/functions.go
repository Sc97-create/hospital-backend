package migration

import (
	"hospital-backend/config"
	"hospital-backend/internal/bedmanagement"
	"hospital-backend/internal/department"
	"hospital-backend/internal/employee"
	"hospital-backend/internal/jwt"
	"hospital-backend/internal/license"
	"hospital-backend/internal/modules"
	"hospital-backend/internal/organisation"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/roles"
	"log"
)

func Migrate() (err error) {
	err = config.PostgreClient.AutoMigrate(&organisation.Organisation{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = config.PostgreClient.AutoMigrate(&license.License{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = config.PostgreClient.AutoMigrate(&employee.User{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = config.PostgreClient.AutoMigrate(&jwt.RefreshToken{})
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	err = config.PostgreClient.AutoMigrate(&patient.Patient{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = config.PostgreClient.AutoMigrate(&department.Department{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = config.PostgreClient.AutoMigrate(&roles.Role{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = config.PostgreClient.AutoMigrate(&permissions.Permission{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = config.PostgreClient.AutoMigrate(&rolepermissions.RolePermission{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = config.PostgreClient.AutoMigrate(&modules.Modules{})
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = bedmanagement.Migrate(config.PostgreClient)
	if err != nil {
		log.Fatalf("%v", err)
	}

	return

}
