package prescription

import (
	"context"

	"gorm.io/gorm"
)

type PrescItemsRepo interface {
	AddItems(db *gorm.DB, edicine []PrescriptionItems) error
<<<<<<< HEAD
	GetMedicineIDsByPrescriptionID(db *gorm.DB, prescriptionID string) ([]string, error)
	GetItemsByPrescriptionID(query string, cond ...any) ([]MixedPrescriptionItem, error)
	GetPrescriptionItemByID(id string) (PrescriptionItems, error)
	UpdatePrescriptionItem(item PrescriptionItems) error
	GetTotalCountByPrescID(prescriptionID string) (int64, error)
	FindMedicineInfoByPID(ctx context.Context, query string, args ...any) ([]MedicineDetInfo, error)
	GetQtyInfoByMed(prescriptionID string) ([]PrescriptionItems, error)
	UpdateDispenseItemQty(tx *gorm.DB, query string, prescriptionItemID string, dispensedQty int64) error
	UpdatePrescriptionItemStatus(tx *gorm.DB, item PrescriptionItems) error
=======
	GetItemsByPrescriptionID(query string, cond ...any) ([]MixedPrescriptionItem, error)
	//UpdatePrescriptionItem(medicine PrescriptionItems) (err error)
	GetTotalCountByPrescID(prescriptionID string) (int64, error)
	FindMedicineInfoByPID(ctx context.Context, query string, args ...any) ([]MedicineDetInfo, error)
>>>>>>> f944f9f (billing.md added details, medicineinventory.md, code changes.)
}

func (pdb *PrescriptionDB) AddItems(db *gorm.DB, medicine []PrescriptionItems) error {
	err := db.CreateInBatches(medicine, len(medicine)).Error
	if err != nil {
		return err
	}
	return nil
}
<<<<<<< HEAD
func (pdb *PrescriptionDB) GetMedicineIDsByPrescriptionID(db *gorm.DB, prescriptionID string) ([]string, error) {
	var medicineIDs []string
	err := db.Model(&PrescriptionItems{}).
		Where("prescription_id = ?", prescriptionID).
		Pluck("medicine_id", &medicineIDs).Error
	if err != nil {
		return nil, err
	}
	return medicineIDs, nil
}
=======
>>>>>>> f944f9f (billing.md added details, medicineinventory.md, code changes.)
func (pdb *PrescriptionDB) GetItemsByPrescriptionID(query string, cond ...any) ([]MixedPrescriptionItem, error) {
	var prescriptionItems []MixedPrescriptionItem
	err := pdb.db.Raw(query, cond...).Find(&prescriptionItems).Error
	if err != nil {
		return nil, err
	}
	return prescriptionItems, nil
}
<<<<<<< HEAD
func (pdb *PrescriptionDB) GetPrescriptionItemByID(id string) (PrescriptionItems, error) {
	var item PrescriptionItems
	err := pdb.db.Where("id = ?", id).First(&item).Error
	if err != nil {
		return PrescriptionItems{}, err
	}
	return item, nil
}
func (pdb *PrescriptionDB) UpdatePrescriptionItem(item PrescriptionItems) error {
	return pdb.db.Model(&PrescriptionItems{}).
		Where("id = ?", item.ID).
		Select("medicine_id", "frequency", "quantity", "duration_day", "duration_type", "food_instruction", "updated_at").
		Updates(item).Error
}
=======
>>>>>>> f944f9f (billing.md added details, medicineinventory.md, code changes.)
func (pdb *PrescriptionDB) GetTotalCountByPrescID(prescriptionID string) (count int64, err error) {
	err = pdb.db.Model(&PrescriptionItems{}).Where("prescription_id=?", prescriptionID).Count(&count).Error
	if err != nil {
		return
	}
	return
}
func (pdb *PrescriptionDB) FindMedicineInfoByPID(ctx context.Context, query string, args ...any) ([]MedicineDetInfo, error) {
	var medicineInfo []MedicineDetInfo
	err := pdb.db.WithContext(ctx).Raw(query, args...).Find(&medicineInfo).Error
	if err != nil {
		return nil, err
	}
	return medicineInfo, nil
}
<<<<<<< HEAD
func (pdb *PrescriptionDB) GetQtyInfoByMed(prescriptionID string) ([]PrescriptionItems, error) {
	var prescriptionItems []PrescriptionItems
	err := pdb.db.Model(&PrescriptionItems{}).Where("prescription_id=?", prescriptionID).Select("id", "medicine_id", "quantity", "balance_after_dispense").Find(&prescriptionItems).Error
	if err != nil {
		return nil, err
	}
	return prescriptionItems, nil
}
func (pdb *PrescriptionDB) UpdateDispenseItemQty(tx *gorm.DB, query string, prescriptionItemID string, dispensedQty int64) error {
	err := tx.Raw(query, prescriptionItemID, dispensedQty).Error
	if err != nil {
		return err
	}
	return nil
}
func (pdb *PrescriptionDB) UpdatePrescriptionItemStatus(tx *gorm.DB, item PrescriptionItems) error {
	err := tx.Model(&PrescriptionItems{}).Where("id=?", item.ID).Updates(item).Error
	if err != nil {
		return err
	}
	return nil
}
=======
>>>>>>> f944f9f (billing.md added details, medicineinventory.md, code changes.)
