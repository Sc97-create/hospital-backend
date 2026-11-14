package department

import (
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DepartmentRepository interface {
	Create(log *zap.Logger, tx *gorm.DB, dept *Department) error
	FindDeptID(log *zap.Logger, organisationID string) (string, error)
	BatchInsert(log *zap.Logger, tx *gorm.DB, dept []Department) error
	FindMany(log *zap.Logger, organisationID string, limit int, skip int) ([]Department, error)
	Count(log *zap.Logger, organisationID string) (int64, error)
	FindDeptByName(log *zap.Logger, organisationID string, name string) (Department, error)
	FindByID(log *zap.Logger, id string) (Department, error)
	LinkByNames(log *zap.Logger, tx *gorm.DB, organisationID string, names []string) error
}

func (Deptdb *DepartmentDB) Create(log *zap.Logger, tx *gorm.DB, dept *Department) (err error) {
	log = ensureLog(log)
	err = Deptdb.DB.Create(&dept).Error
	if err != nil {
		logDBError(log, "Create", err)
		return
	}
	return
}

func (DeptDb *DepartmentDB) FindDeptID(log *zap.Logger, organisationID string) (departmentID string, err error) {
	log = ensureLog(log)
	var row struct {
		ID string `gorm:"column:id"`
	}
	err = DeptDb.DB.Table("departments AS d").
		Select("d.id").
		Joins("join organisation_departments od on od.department_id = d.id").
		Where("od.organisation_id = ?", organisationID).
		Limit(1).
		Scan(&row).Error
	departmentID = row.ID
	if err != nil {
		logDBError(log, "FindDeptID", err)
		return
	}
	return
}

func (DeptDb *DepartmentDB) BatchInsert(log *zap.Logger, tx *gorm.DB, depts []Department) (err error) {
	log = ensureLog(log)
	err = tx.CreateInBatches(depts, len(depts)).Error
	if err != nil {
		logDBError(log, "BatchInsert", err)
		return
	}
	return
}

func (DeptDb *DepartmentDB) FindMany(log *zap.Logger, organisationID string, limit int, skip int) ([]Department, error) {
	log = ensureLog(log)
	query := `select d.id, d.name
		from departments d
		join organisation_departments od on od.department_id = d.id
		where od.organisation_id=$1 and d.name != $2
		limit $3 offset $4`
	var departments []Department
	err := DeptDb.DB.Raw(query, organisationID, DefaultDeptAdmin, limit, skip).Scan(&departments).Error
	if err != nil {
		logDBError(log, "FindMany", err)
		return nil, err
	}
	return departments, nil
}

func (DeptDb *DepartmentDB) Count(log *zap.Logger, organisationID string) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := DeptDb.DB.Table("departments AS d").
		Joins("join organisation_departments od on od.department_id = d.id").
		Where("od.organisation_id = ? AND d.name != ?", organisationID, DefaultDeptAdmin).
		Count(&count).Error
	if err != nil {
		logDBError(log, "Count", err)
		return 0, err
	}
	return count, nil
}

func (DeptDb *DepartmentDB) FindDeptByName(log *zap.Logger, organisationID string, name string) (Department, error) {
	log = ensureLog(log)
	var dept Department
	query := `select id, name, facility_type_id from departments where name=$1`
	err := DeptDb.DB.Raw(query, name).Scan(&dept).Error
	if err != nil {
		logDBError(log, "FindDeptByName", err)
		return Department{}, err
	}
	return dept, nil
}

func (DeptDb *DepartmentDB) FindByID(log *zap.Logger, id string) (Department, error) {
	log = ensureLog(log)
	var dept Department
	query := `select id,name from departments where id=$1`
	err := DeptDb.DB.Raw(query, id).Scan(&dept).Error
	if err != nil {
		logDBError(log, "FindByID", err)
		return Department{}, err
	}
	return dept, nil
}

func (DeptDb *DepartmentDB) LinkByNames(log *zap.Logger, tx *gorm.DB, organisationID string, names []string) error {
	log = ensureLog(log)
	if len(names) == 0 || strings.TrimSpace(organisationID) == "" {
		return nil
	}
	err := deptDB(DeptDb.DB, tx).Exec(`
		INSERT INTO organisation_departments (organisation_id, department_id, status)
		SELECT ?, d.id, 'active'
		FROM departments d
		WHERE d.name IN ?
		ON CONFLICT (organisation_id, department_id) DO NOTHING
	`, organisationID, names).Error
	if err != nil {
		logDBError(log, "LinkByNames", err)
	}
	return err
}

func deptDB(db, tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return db
}
