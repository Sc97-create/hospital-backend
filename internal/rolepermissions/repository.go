package rolepermissions

import (
	"errors"

	"hospital-backend/internal/rolepermissions/dto"

	"gorm.io/gorm"
)

type RolePermissionRepo interface {
	Create(rolePermission *RolePermission) error
	BatchCreate(tx *gorm.DB, rolePermissions []RolePermission) error
	FindById(id string) (*RolePermission, error)
	FindModulePermissionsByRoleID(roleID string) ([]dto.ModulePermissionRow, error)
	IsAdminRole(roleID string) (bool, error)
}

func (RPerm *RolePermissionDb) Create(rolePermission *RolePermission) error {
	return RPerm.DB.Create(rolePermission).Error
}
func (RPerm *RolePermissionDb) BatchCreate(tx *gorm.DB, rolePermissions []RolePermission) (err error) {
	db := RPerm.DB
	if tx != nil {
		db = tx
	}
	for i := range rolePermissions {
		if err = createRolePermission(db, &rolePermissions[i]); err != nil {
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
func (RPerm *RolePermissionDb) FindById(id string) (*RolePermission, error) {
	var rolePermission RolePermission
	return &rolePermission, RPerm.DB.First(&rolePermission, id).Error
}

func (RPerm *RolePermissionDb) FindModulePermissionsByRoleID(roleID string) ([]dto.ModulePermissionRow, error) {
	query := `SELECT mo.name AS module_name, array_agg(pr.name) AS permission_names
		FROM role_permissions rp
		JOIN modules mo ON rp.module_id = mo.id
		JOIN permissions pr ON rp.permission_id = pr.id
		WHERE rp.role_id = ?
		GROUP BY mo.id, mo.name`
	var rows []dto.ModulePermissionRow
	err := RPerm.DB.Raw(query, roleID).Scan(&rows).Error
	return rows, err
}

func (RPerm *RolePermissionDb) IsAdminRole(roleID string) (bool, error) {
	var isAdmin bool
	err := RPerm.DB.Raw(
		`SELECT EXISTS (
			SELECT 1 FROM role_permissions WHERE role_id = ? AND is_admin = true
		)`,
		roleID,
	).Scan(&isAdmin).Error
	return isAdmin, err
}
