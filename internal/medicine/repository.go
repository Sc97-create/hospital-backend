package medicine

type MedicineRepository interface {
	Create(*Medicine) (err error)
	FindOne(id string) (*Medicine, error)
	FindMany(limit int, offset int) ([]Medicine, error)
	Update(id string, update map[string]interface{}) error
}

func (MRepo *MedicineRepo) Create(M *Medicine) (err error) {
	err = MRepo.db.Create(&M).Error
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
func (MRepo *MedicineRepo) FindMany(limit int, offset int) (Med []Medicine, err error) {
	err = MRepo.db.Find(&Med, "limit ? offset ?", limit, offset).Error
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
