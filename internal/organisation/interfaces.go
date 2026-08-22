package organisation

import (
	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/roles"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PermissionCatalogLookup interface {
	FindMany() ([]modules.Modules, []permissions.Permission, error)
}

type LicenseCreator interface {
	CreateLicenseSrv(log *zap.Logger, tx *gorm.DB, orgname string, planday int, organisationID string, planspan string, issuedAt time.Time) error
}

type RoleSeeder interface {
	InsertMany(tx *gorm.DB, organisationID string) ([]roles.Role, error)
}

type DepartmentSeeder interface {
	InsertMany(tx *gorm.DB, organisationID string) error
}

type RolePermissionSeeder interface {
	InsertMany(tx *gorm.DB, roleArr []roles.Role, permissionsArr []permissions.Permission, modulesArr []modules.Modules, organisationID string) error
}
