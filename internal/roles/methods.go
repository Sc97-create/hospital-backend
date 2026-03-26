package roles

import "gorm.io/gorm"

type RoleDB struct {
	DB *gorm.DB
}

func NewRoleRepo(db *gorm.DB) *RoleDB {
	return &RoleDB{DB: db}
}
