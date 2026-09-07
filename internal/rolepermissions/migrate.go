package rolepermissions

import "gorm.io/gorm"

// RelaxNullableColumns drops NOT NULL on columns that can be empty for admin grants.
// GORM AutoMigrate does not remove existing NOT NULL constraints.
func RelaxNullableColumns(db *gorm.DB) error {
	stmts := []string{
		`ALTER TABLE role_permissions ALTER COLUMN permission_id DROP NOT NULL`,
		`ALTER TABLE role_permissions ALTER COLUMN module_id DROP NOT NULL`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
