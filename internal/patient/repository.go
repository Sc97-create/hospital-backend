package patient

import (
	"errors"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PatientRepository interface {
	Create(log *zap.Logger, record *Patient) error
	ReadMany(log *zap.Logger, query string, args ...any) ([]Patient, error)
	ReadOne(log *zap.Logger, patientID string) (Patient, error)
	Count(log *zap.Logger, query string, args ...any) (int64, error)
	ReadOneWithOrganisationID(log *zap.Logger, query string, args ...any) (map[string]interface{}, error)
}

func (p *PatientRepo) Create(log *zap.Logger, record *Patient) error {
	log = ensureLog(log)
	err := p.db.Create(&record).Error
	if err != nil {
		log.Error("patient repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (p *PatientRepo) ReadMany(log *zap.Logger, query string, args ...any) (patients []Patient, err error) {
	log = ensureLog(log)
	err = p.db.Raw(query, args...).Scan(&patients).Error
	if err != nil {
		log.Error("patient repo error", zap.String("op", "ReadMany"), zap.Error(err))
		return
	}
	if patients == nil {
		patients = []Patient{}
	}
	return
}

func (p *PatientRepo) ReadOne(log *zap.Logger, id string) (patient Patient, err error) {
	log = ensureLog(log)
	err = p.db.First(&patient, "id=?", id).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("patient repo error", zap.String("op", "ReadOne"), zap.Error(err))
		}
		return
	}
	return
}

func (p *PatientRepo) Count(log *zap.Logger, query string, args ...any) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := p.db.Raw(query, args...).Scan(&count).Error
	if err != nil {
		log.Error("patient repo error", zap.String("op", "Count"), zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (p *PatientRepo) ReadOneWithOrganisationID(log *zap.Logger, query string, args ...any) (patient map[string]interface{}, err error) {
	log = ensureLog(log)
	err = p.db.Raw(query, args...).Scan(&patient).Error
	if err != nil {
		log.Error("patient repo error", zap.String("op", "ReadOneWithOrganisationID"), zap.Error(err))
		return nil, err
	}
	return patient, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}
