package db

import (
	"strings"

	"gorm.io/gorm"
)

// EnsureForeignKeys runs ALTER TABLE ADD CONSTRAINT statements idempotently.
// Errors that indicate the constraint already exists are ignored.
func EnsureForeignKeys(gdb *gorm.DB, stmts []string) error {
	for _, sql := range stmts {
		if err := gdb.Exec(sql).Error; err != nil && !IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

// IsAlreadyExists reports whether err is a Postgres "already exists" failure.
func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists")
}
