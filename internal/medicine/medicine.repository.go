package medicine

import "gorm.io/gorm"

type MedicineRepository interface {
	CreateInBatches(db *gorm.DB, M []Medicine) (err error)
	FindOne(id string) (*Medicine, error)
	FindMany(query string, args ...any) ([]Medicine, error)
	Update(id string, update map[string]interface{}) error
	FindNamesByIds([]string) ([]Medicine, error)
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
