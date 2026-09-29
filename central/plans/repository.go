package plans

import (
	"strings"

	"hospital-backend/pkg/constants"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PlanRepo struct {
	db *gorm.DB
}

func NewPlanRepo(db *gorm.DB) *PlanRepo {
	return &PlanRepo{db: db}
}

type PlanRepository interface {
	GetByID(log *zap.Logger, planID string) (Plan, error)
	GetByName(log *zap.Logger, name string) (Plan, error)
	ListActive(log *zap.Logger) ([]Plan, error)
}

func ensureLog(log *zap.Logger) *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}
	return log
}

func (r *PlanRepo) GetByID(log *zap.Logger, planID string) (Plan, error) {
	log = ensureLog(log)
	var plan Plan
	err := r.db.Where("id = ?", planID).First(&plan).Error
	if err != nil {
		log.Error("plan repo error", zap.String("op", "GetByID"), zap.Error(err))
		return Plan{}, err
	}
	return plan, nil
}

func (r *PlanRepo) GetByName(log *zap.Logger, name string) (Plan, error) {
	log = ensureLog(log)
	var plan Plan
	err := r.db.Where("LOWER(name) = ?", strings.ToLower(strings.TrimSpace(name))).First(&plan).Error
	if err != nil {
		log.Error("plan repo error", zap.String("op", "GetByName"), zap.Error(err))
		return Plan{}, err
	}
	return plan, nil
}

func (r *PlanRepo) ListActive(log *zap.Logger) ([]Plan, error) {
	log = ensureLog(log)
	var rows []Plan
	err := r.db.Where("status = ?", constants.StatusActive).Order("monthly_price asc").Find(&rows).Error
	if err != nil {
		log.Error("plan repo error", zap.String("op", "ListActive"), zap.Error(err))
		return nil, err
	}
	return rows, nil
}
