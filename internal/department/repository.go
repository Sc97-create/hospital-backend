package department

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DepartmentRepository interface {
	Create(*gorm.DB, *Department) error
	FindDeptID(organisationID string) (string, error)
	BatchInsert(tx *gorm.DB, dept []Department, size int) error
	FindMany(limit int, skip int) ([]Department, error)
}

func (Deptdb *DepartmentDB) Create(tx *gorm.DB, dept *Department) (err error) {
	err = Deptdb.DB.Create(&dept).Error
	if err != nil {
		return
	}
	return
}
func (DeptDb *DepartmentDB) FindDeptID(organisationID string) (departmentID string, err error) {
	err = DeptDb.DB.Select("id").Where("organisation_id=?", organisationID).First(departmentID).Error
	if err != nil {
		return
	}
	return
}
func (DeptDb *DepartmentDB) BatchInsert(tx *gorm.DB, depts []Department, size int) error {
	return DeptDb.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).CreateInBatches(depts, size).Error
}
func (DeptDb *DepartmentDB) FindMany(limit int, skip int) ([]Department, error) {
	query := `select id,name from departments where is_default=true`
	var departments []Department
	err := DeptDb.DB.Raw(query).Limit(limit).Offset(skip).Scan(&departments).Error
	if err != nil {
		return nil, err
	}
	return departments, nil
}
