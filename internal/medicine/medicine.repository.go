package medicine

import (
	"hospital-backend/internal/medicine/dto"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MedicineRepository interface {
	CreateInBatches(log *zap.Logger, db *gorm.DB, M []Medicine) (err error)
	FindOne(log *zap.Logger, id string) (*Medicine, error)
	FindMany(log *zap.Logger, query string, args ...any) ([]Medicine, error)
	SearchMedicine(log *zap.Logger, name string, pattern string, organisationID string) ([]dto.SearchMedicineItem, error)
	Update(log *zap.Logger, id string, update map[string]interface{}) error
	FindNamesByIds(log *zap.Logger, ids []string) ([]Medicine, error)
	GetMedicineByID(log *zap.Logger, medicineID string) (medicine Medicine, err error)
}

func (MRepo *MedicineRepo) CreateInBatches(log *zap.Logger, db *gorm.DB, M []Medicine) (err error) {
	log = ensureLog(log)
	if len(M) == 0 {
		return nil
	}
	err = db.CreateInBatches(&M, len(M)).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "CreateInBatches"), zap.Error(err))
		return
	}
	return
}

func (MRepo *MedicineRepo) FindOne(log *zap.Logger, id string) (*Medicine, error) {
	log = ensureLog(log)
	var Med Medicine
	err := MRepo.db.First(&Med, "id=?", id).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("medicine repo error", zap.String("op", "FindOne"), zap.Error(err))
		}
		return nil, err
	}
	return &Med, nil
}

func (MRepo *MedicineRepo) FindMany(log *zap.Logger, query string, args ...any) (Med []Medicine, err error) {
	log = ensureLog(log)
	err = MRepo.db.Model(&Medicine{}).Select("id,name,form,strength").Where(query, args...).Find(&Med).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "FindMany"), zap.Error(err))
		return
	}
	return
}

func (MRepo *MedicineRepo) SearchMedicine(log *zap.Logger, name string, pattern string, organisationID string) ([]dto.SearchMedicineItem, error) {
	log = ensureLog(log)
	var results []dto.SearchMedicineItem
	sql := `
		SELECT
			m.id,
			m.name,
			m.form,
			m.strength,
			m.hsn_code,
			m.reorder_level,
			m.max_stock_target,
			COALESCE((
				SELECT mi.shelf_location
				FROM medicine_inventories mi
				WHERE mi.medicine_id = m.id
				  AND mi.organisation_id = $3
				ORDER BY mi.created_at DESC
				LIMIT 1
			), '') AS shelf_location,
			COALESCE((
				SELECT SUM(mi.current_stock_units)
				FROM medicine_inventories mi
				WHERE mi.medicine_id = m.id
				  AND mi.current_stock_units > 0
				  AND mi.organisation_id = $3
			), 0) AS current_stock_units
		FROM medicines m
		WHERE ($1 = '' OR m.name ILIKE $2 OR m.code ILIKE $2)
		  AND m.organisation_id = $3`
	err := MRepo.db.Raw(sql, name, pattern, organisationID).Scan(&results).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "SearchMedicine"), zap.Error(err))
		return nil, err
	}
	if results == nil {
		results = []dto.SearchMedicineItem{}
	}
	return results, nil
}

func (MRepo *MedicineRepo) Update(log *zap.Logger, id string, updates map[string]interface{}) (err error) {
	log = ensureLog(log)
	err = MRepo.db.Model(&Medicine{}).Where("id=?", id).Updates(updates).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "Update"), zap.Error(err))
		return
	}
	return
}

func (MRepo *MedicineRepo) FindNamesByIds(log *zap.Logger, ids []string) (Med []Medicine, err error) {
	log = ensureLog(log)
	err = MRepo.db.Model(&Medicine{}).Select("id,name").Where("id IN ?", ids).Find(&Med).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "FindNamesByIds"), zap.Error(err))
		return
	}
	return
}

func (MRepo *MedicineRepo) GetMedicineByID(log *zap.Logger, medicineID string) (medicine Medicine, err error) {
	log = ensureLog(log)
	err = MRepo.db.Where("id=?", medicineID).First(&medicine).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("medicine repo error", zap.String("op", "GetMedicineByID"), zap.Error(err))
		}
		return
	}
	return
}

