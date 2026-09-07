package department

import (
	"gorm.io/gorm"
)

type DepartmentRepository interface {
	Create(*gorm.DB, *Department) error
	FindDeptID(organisationID string) (string, error)
	BatchInsert(tx *gorm.DB, dept []Department) error
	FindMany(organisationID string, limit int, skip int) ([]Department, error)
	Count(organisationID string) (int64, error)
	FindDeptByName(organisationID string, name string) (Department, error)
	FindByID(id string) (Department, error)
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
func (DeptDb *DepartmentDB) BatchInsert(tx *gorm.DB, depts []Department) (err error) {
	err = tx.CreateInBatches(depts, len(depts)).Error
	if err != nil {
		return
	}
	return
}
func (DeptDb *DepartmentDB) FindMany(organisationID string, limit int, skip int) ([]Department, error) {
	query := `select id,name from departments where organisation_id=$1 and name != $2 limit $3 offset $4`
	var departments []Department
	err := DeptDb.DB.Raw(query, organisationID, DefaultDeptAdmin, limit, skip).Scan(&departments).Error
	if err != nil {
		return nil, err
	}
	return departments, nil
}

func (DeptDb *DepartmentDB) Count(organisationID string) (int64, error) {
	var count int64
	err := DeptDb.DB.Model(&Department{}).Where("organisation_id=? AND name != ?", organisationID, DefaultDeptAdmin).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
func (DeptDb *DepartmentDB) FindDeptByName(organisationID string, name string) (Department, error) {
	var dept Department
	query := `select id,name from departments where organisation_id=$1 and name=$2`
	err := DeptDb.DB.Raw(query, organisationID, name).Scan(&dept).Error
	if err != nil {
		return Department{}, err
	}
	return dept, nil
}

func (DeptDb *DepartmentDB) FindByID(id string) (Department, error) {
	var dept Department
	query := `select id,name from departments where id=$1`
	err := DeptDb.DB.Raw(query, id).Scan(&dept).Error
	if err != nil {
		return Department{}, err
	}
	return dept, nil
}
