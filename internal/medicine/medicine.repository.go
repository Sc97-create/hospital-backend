package medicine

import (
	"hospital-backend/internal/medicine/dto"

	"gorm.io/gorm"
)

type MedicineRepository interface {
	CreateInBatches(db *gorm.DB, M []Medicine) (err error)
	FindOne(id string) (*Medicine, error)
	FindMany(query string, args ...any) ([]Medicine, error)
	SearchMedicine(name string, pattern string, organisationID string) ([]dto.SearchMedicineItem, error)
	Update(id string, update map[string]interface{}) error
	FindNamesByIds([]string) ([]Medicine, error)
	GetMedicineByID(medicineID string) (medicine Medicine, err error)
}

func (MRepo *MedicineRepo) CreateInBatches(db *gorm.DB, M []Medicine) (err error) {
	err = db.CreateInBatches(&M, len(M)).Error
	if err != nil {
		return
	}
	return
}
func (MRepo *MedicineRepo) FindOne(id string) (*Medicine, error) {
	var Med Medicine
	err := MRepo.db.First(&Med, "id=?", id).Error
	if err != nil {
		return nil, err
	}
	return &Med, nil
}
func (MRepo *MedicineRepo) FindMany(query string, args ...any) (Med []Medicine, err error) {
	err = MRepo.db.Model(&Medicine{}).Select("id,name,form,strength").Where(query, args...).Find(&Med).Error
	if err != nil {
		return
	}
	return
}
func (MRepo *MedicineRepo) SearchMedicine(name string, pattern string, organisationID string) ([]dto.SearchMedicineItem, error) {
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
		return nil, err
	}
	if results == nil {
		results = []dto.SearchMedicineItem{}
	}
	return results, nil
}
func (Mrepo *MedicineRepo) Update(id string, updates map[string]interface{}) (err error) {
	err = Mrepo.db.Model(&Medicine{}).Where("id=?", id).Updates(updates).Error
	if err != nil {
		return
	}
	return
}
func (MRepo *MedicineRepo) FindNamesByIds(ids []string) (Med []Medicine, err error) {
	err = MRepo.db.Model(&Medicine{}).Select("id,name").Where("id IN ?", ids).Find(&Med).Error
	if err != nil {
		return
	}
	return
}
func (MRepo *MedicineRepo) GetMedicineByID(medicineID string) (medicine Medicine, err error) {
	err = MRepo.db.Where("id=?", medicineID).First(&medicine).Error
	if err != nil {
		return
	}
	return
}
