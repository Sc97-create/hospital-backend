package labservice

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LabTestTemplateRepository persists lab test template rows.
type LabTestTemplateRepository interface {
	Create(log *zap.Logger, template *LabTestTemplate) error
	FindByID(log *zap.Logger, id string) (LabTestTemplate, error)
	FindMany(log *zap.Logger, query string, args ...any) ([]LabTestTemplate, error)
	Count(log *zap.Logger, query string, args ...any) (int64, error)
}

type labTestTemplateDB struct {
	db *gorm.DB
}

func NewLabTestTemplateRepo(db *gorm.DB) LabTestTemplateRepository {
	return &labTestTemplateDB{db: db}
}

func (r *labTestTemplateDB) Create(log *zap.Logger, template *LabTestTemplate) error {
	log = ensureLog(log)
	if err := r.db.Create(template).Error; err != nil {
		log.Error("lab test template repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (r *labTestTemplateDB) FindByID(log *zap.Logger, id string) (LabTestTemplate, error) {
	log = ensureLog(log)
	var row LabTestTemplate
	err := r.db.Model(&LabTestTemplate{}).Where("id = ?", id).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("lab test template repo error", zap.String("op", "FindByID"), zap.Error(err))
		}
		return LabTestTemplate{}, err
	}
	return row, nil
}

func (r *labTestTemplateDB) FindMany(log *zap.Logger, query string, args ...any) ([]LabTestTemplate, error) {
	log = ensureLog(log)
	var rows []LabTestTemplate
	if err := r.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		log.Error("lab test template repo error", zap.String("op", "FindMany"), zap.Error(err))
		return nil, err
	}
	if rows == nil {
		rows = []LabTestTemplate{}
	}
	return rows, nil
}

func (r *labTestTemplateDB) Count(log *zap.Logger, query string, args ...any) (int64, error) {
	log = ensureLog(log)
	var count int64
	if err := r.db.Raw(query, args...).Scan(&count).Error; err != nil {
		log.Error("lab test template repo error", zap.String("op", "Count"), zap.Error(err))
		return 0, err
	}
	return count, nil
}
