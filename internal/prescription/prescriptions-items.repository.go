package prescription

import (
	"context"

	"gorm.io/gorm"
)

type PrescItemsRepo interface {
	AddItems(db *gorm.DB, edicine []PrescriptionItems) error
	GetItemsByPrescriptionID(query string, cond ...any) ([]MixedPrescriptionItem, error)
	//UpdatePrescriptionItem(medicine PrescriptionItems) (err error)
	GetTotalCountByPrescID(prescriptionID string) (int64, error)
	FindMedicineInfoByPID(ctx context.Context, query string, args ...any) ([]MedicineDetInfo, error)
}

func (pdb *PrescriptionDB) AddItems(db *gorm.DB, medicine []PrescriptionItems) error {
	err := db.CreateInBatches(medicine, len(medicine)).Error
	if err != nil {
		return err
	}
	return nil
}
func (pdb *PrescriptionDB) GetItemsByPrescriptionID(query string, cond ...any) ([]MixedPrescriptionItem, error) {
	var prescriptionItems []MixedPrescriptionItem
	err := pdb.db.Raw(query, cond...).Find(&prescriptionItems).Error
	if err != nil {
		return nil, err
	}
	return prescriptionItems, nil
}
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
