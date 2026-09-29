package rolepermissions

import (
	"errors"

	"hospital-backend/internal/rolepermissions/dto"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RolePermissionRepo interface {
	Create(log *zap.Logger, rolePermission *RolePermission) error
	BatchCreate(log *zap.Logger, tx *gorm.DB, rolePermissions []RolePermission) error
	FindById(log *zap.Logger, id string) (*RolePermission, error)
	FindModulePermissionsByRoleID(log *zap.Logger, roleID string) ([]dto.ModulePermissionRow, error)
	IsAdminRole(log *zap.Logger, roleID string) (bool, error)
}

func (RPerm *RolePermissionDb) Create(log *zap.Logger, rolePermission *RolePermission) error {
	log = ensureLog(log)
	err := RPerm.DB.Create(rolePermission).Error
	if err != nil {
		logDBError(log, "Create", err)
	}
	return err
}

func (RPerm *RolePermissionDb) BatchCreate(log *zap.Logger, tx *gorm.DB, rolePermissions []RolePermission) (err error) {
	log = ensureLog(log)
	db := RPerm.DB
	if tx != nil {
		db = tx
	}
	for i := range rolePermissions {
		if err = createRolePermission(db, &rolePermissions[i]); err != nil {
			logDBError(log, "BatchCreate", err)
			return err
		}
	}
	if len(rolePermissions) == 0 {
		return errors.New("no role permissions inserted")
	}
	return nil
}

func createRolePermission(db *gorm.DB, rp *RolePermission) error {
	if rp.RoleID == "" {
		return nil
	}
	q := db.Model(&RolePermission{})
	if rp.PermissionID == nil {
		q = q.Omit("PermissionID")
	}
	if rp.ModuleID == nil {
		q = q.Omit("ModuleID")
	}
	return q.Create(rp).Error
}

func (RPerm *RolePermissionDb) FindById(log *zap.Logger, id string) (*RolePermission, error) {
	log = ensureLog(log)
	var rolePermission RolePermission
	err := RPerm.DB.First(&rolePermission, id).Error
	if err != nil {
		logDBError(log, "FindById", err)
	}
	return &rolePermission, err
}

func (RPerm *RolePermissionDb) FindModulePermissionsByRoleID(log *zap.Logger, roleID string) ([]dto.ModulePermissionRow, error) {
	log = ensureLog(log)
	query := `SELECT mo.name AS module_name, array_agg(pr.name) AS permission_names
		FROM role_permissions rp
		JOIN modules mo ON rp.module_id = mo.id
		JOIN permissions pr ON rp.permission_id = pr.id
		WHERE rp.role_id = ?
		GROUP BY mo.id, mo.name`
	var rows []dto.ModulePermissionRow
	err := RPerm.DB.Raw(query, roleID).Scan(&rows).Error
	if err != nil {
		logDBError(log, "FindModulePermissionsByRoleID", err)
	}
	return rows, err
}

func (RPerm *RolePermissionDb) IsAdminRole(log *zap.Logger, roleID string) (bool, error) {
	log = ensureLog(log)
	var isAdmin bool
	err := RPerm.DB.Raw(
		`SELECT EXISTS (
			SELECT 1 FROM role_permissions WHERE role_id = ? AND is_admin = true
		)`,
		roleID,
	).Scan(&isAdmin).Error
	if err != nil {
		logDBError(log, "IsAdminRole", err)
	}
	return isAdmin, err
}
