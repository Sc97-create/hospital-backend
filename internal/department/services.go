package department

import (
	"time"

	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DepartmentService struct {
	DepRepo DepartmentRepository
}

func NewDepartmentService(repo DepartmentRepository) *DepartmentService {
	return &DepartmentService{DepRepo: repo}
}

func (DeptService *DepartmentService) FindMany(log *zap.Logger, organisationID string, limit int, skip int) ([]Department, int64, error) {
	log = ensureLog(log)
	departments, err := DeptService.DepRepo.FindMany(log, organisationID, limit, skip)
	if err != nil {
		log.Error("department list failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_list"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrDepartmentsFetchFailed
	}
	total, err := DeptService.DepRepo.Count(log, organisationID)
	if err != nil {
		log.Error("department list failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrDepartmentsFetchFailed
	}
	return departments, total, nil
}

func (DeptService *DepartmentService) InsertMany(log *zap.Logger, tx *gorm.DB, organisationID string) error {
	log = ensureLog(log)
	deptArray := DeptService.createDeptArray(organisationID)
	err := DeptService.DepRepo.BatchInsert(log, tx, deptArray)
	if err != nil {
		log.Error("department seed failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_insert"),
			zap.Error(err),
		)
		return err
	}
	log.Info("department seed success",
		zap.String("organisation_id", organisationID),
		zap.Int("count", len(deptArray)),
	)
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

func (DeptService *DepartmentService) FindDeptByName(log *zap.Logger, organisationID string, name string) (Department, error) {
	log = ensureLog(log)
	dept, err := DeptService.DepRepo.FindDeptByName(log, organisationID, name)
	if err != nil {
		log.Error("department lookup failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return Department{}, err
	}
	return dept, nil
}

func (DeptService *DepartmentService) FindByID(log *zap.Logger, id string) (Department, error) {
	log = ensureLog(log)
	dept, err := DeptService.DepRepo.FindByID(log, id)
	if err != nil {
		log.Error("department lookup failed",
			zap.String("department_id", id),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return Department{}, err
	}
	return dept, nil
}
