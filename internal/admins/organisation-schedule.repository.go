package admins

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrganisationScheduleRepository interface {
	Create(log *zap.Logger, schedule *OrganisationSchedule) error
	GetByOrganisationID(log *zap.Logger, query string, cond ...any) (OrganisationSchedule, error)
}

func (r *CommonDB) Create(log *zap.Logger, schedule *OrganisationSchedule) error {
	log = ensureLog(log)
	err := r.db.Create(schedule).Error
	if err != nil {
		log.Error("organisation schedule repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (r *CommonDB) GetByOrganisationID(log *zap.Logger, query string, cond ...any) (OrganisationSchedule, error) {
	log = ensureLog(log)
	var orgSched OrganisationSchedule
	err := r.db.Raw(query, cond...).Scan(&orgSched).Error
	if err != nil {
		log.Error("organisation schedule repo error", zap.String("op", "GetByOrganisationID"), zap.Error(err))
		return OrganisationSchedule{}, err
	}
	if orgSched.ID == "" {
		return OrganisationSchedule{}, gorm.ErrRecordNotFound
	}
	return orgSched, nil
}
