package department

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DepartmentService struct {
	DepRepo DepartmentRepository
}

func NewDepartmentService(repo DepartmentRepository) *DepartmentService {
	return &DepartmentService{DepRepo: repo}
}

func (DeptService *DepartmentService) FindMany(organisationID string, limit int, skip int) ([]Department, int64, error) {
	departments, err := DeptService.DepRepo.FindMany(organisationID, limit, skip)
	if err != nil {
		return nil, 0, err
	}
	total, err := DeptService.DepRepo.Count(organisationID)
	if err != nil {
		return nil, 0, err
	}
	return departments, total, nil
}
func (DeptService *DepartmentService) InsertMany(tx *gorm.DB, organisationID string) error {
	deptArray := DeptService.createDeptArray(organisationID)
	err := DeptService.DepRepo.BatchInsert(tx, deptArray)
	if err != nil {
		return err
	}
	return nil
}
func (DeptService *DepartmentService) createDeptArray(organisationID string) []Department {
	var defaultDepartments []Department
	for _, each := range DefaultDeptArr {
		defaultDepartments = append(defaultDepartments, Department{
			ID:             uuid.NewString(),
			Name:           each,
			CreatedAt:      time.Now(),
			OrganisationID: organisationID,
			IsActive:       true,
		})
	}
	return defaultDepartments
}
func (DeptService *DepartmentService) FindDeptByName(organisationID string, name string) (Department, error) {
	return DeptService.DepRepo.FindDeptByName(organisationID, name)
}

func (DeptService *DepartmentService) FindByID(id string) (Department, error) {
	return DeptService.DepRepo.FindByID(id)
}
