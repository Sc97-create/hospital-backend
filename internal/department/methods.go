package department

import "gorm.io/gorm"

type DepartmentDB struct {
	DB *gorm.DB
}

func NewDepartmentRepo(db *gorm.DB) *DepartmentDB {
	return &DepartmentDB{DB: db}
}
