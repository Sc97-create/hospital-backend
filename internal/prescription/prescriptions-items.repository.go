package prescription

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PrescItemsRepo interface {
	AddItems(log *zap.Logger, db *gorm.DB, medicine []PrescriptionItems) error
	GetMedicineIDsByPrescriptionID(log *zap.Logger, db *gorm.DB, prescriptionID string) ([]string, error)
	GetItemsByPrescriptionID(log *zap.Logger, query string, cond ...any) ([]MixedPrescriptionItem, error)
	GetPrescriptionItemByID(log *zap.Logger, id string) (PrescriptionItems, error)
	UpdatePrescriptionItem(log *zap.Logger, item PrescriptionItems) error
	GetTotalCountByPrescID(log *zap.Logger, prescriptionID string) (int64, error)
	FindMedicineInfoByPID(log *zap.Logger, ctx context.Context, query string, args ...any) ([]MedicineDetInfo, error)
	GetPatientByPrescriptionID(log *zap.Logger, query string, prescriptionID string) (MedicineInfoPatientRow, error)
	GetQtyInfoByMed(log *zap.Logger, prescriptionID string) ([]PrescriptionItems, error)
	GetItemStatusesByPrescriptionID(log *zap.Logger, tx *gorm.DB, prescriptionID string) ([]PrescriptionItems, error)
	UpdateDispenseItemQty(log *zap.Logger, tx *gorm.DB, query string, prescriptionItemID string, dispensedQty int64) error
	UpdatePrescriptionItemStatus(log *zap.Logger, tx *gorm.DB, prescriptionItemID string, status string, outOfStock bool) error
}

func (pdb *PrescriptionDB) AddItems(log *zap.Logger, db *gorm.DB, medicine []PrescriptionItems) error {
	log = ensureLog(log)
	err := db.CreateInBatches(medicine, len(medicine)).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "AddItems"), zap.Error(err))
		return err
	}
	return nil
}

func (pdb *PrescriptionDB) GetMedicineIDsByPrescriptionID(log *zap.Logger, db *gorm.DB, prescriptionID string) ([]string, error) {
	log = ensureLog(log)
	var medicineIDs []string
	err := db.Model(&PrescriptionItems{}).
		Where("prescription_id = ?", prescriptionID).
		Pluck("medicine_id", &medicineIDs).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "GetMedicineIDsByPrescriptionID"), zap.Error(err))
		return nil, err
	}
	return medicineIDs, nil
}

func (pdb *PrescriptionDB) GetItemsByPrescriptionID(log *zap.Logger, query string, cond ...any) ([]MixedPrescriptionItem, error) {
	log = ensureLog(log)
	var prescriptionItems []MixedPrescriptionItem
	err := pdb.db.Raw(query, cond...).Find(&prescriptionItems).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "GetItemsByPrescriptionID"), zap.Error(err))
		return nil, err
	}
	return prescriptionItems, nil
}

func (pdb *PrescriptionDB) GetPrescriptionItemByID(log *zap.Logger, id string) (PrescriptionItems, error) {
	log = ensureLog(log)
	var item PrescriptionItems
	err := pdb.db.Where("id = ?", id).First(&item).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("prescription item repo error", zap.String("op", "GetPrescriptionItemByID"), zap.Error(err))
		}
		return PrescriptionItems{}, err
	}
	return item, nil
}

func (pdb *PrescriptionDB) UpdatePrescriptionItem(log *zap.Logger, item PrescriptionItems) error {
	log = ensureLog(log)
	err := pdb.db.Model(&PrescriptionItems{}).
		Where("id = ?", item.ID).
		Select("medicine_id", "frequency", "quantity", "duration_day", "duration_type", "food_instruction", "updated_at").
		Updates(item).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "UpdatePrescriptionItem"), zap.Error(err))
		return err
	}
	return nil
}

func (pdb *PrescriptionDB) GetTotalCountByPrescID(log *zap.Logger, prescriptionID string) (count int64, err error) {
	log = ensureLog(log)
	err = pdb.db.Model(&PrescriptionItems{}).Where("prescription_id=?", prescriptionID).Count(&count).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "GetTotalCountByPrescID"), zap.Error(err))
		return
	}
	return
}

func (pdb *PrescriptionDB) FindMedicineInfoByPID(log *zap.Logger, ctx context.Context, query string, args ...any) ([]MedicineDetInfo, error) {
	log = ensureLog(log)
	var medicineInfo []MedicineDetInfo
	err := pdb.db.WithContext(ctx).Raw(query, args...).Find(&medicineInfo).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "FindMedicineInfoByPID"), zap.Error(err))
		return nil, err
	}
	return medicineInfo, nil
}

func (pdb *PrescriptionDB) GetPatientByPrescriptionID(log *zap.Logger, query string, prescriptionID string) (MedicineInfoPatientRow, error) {
	log = ensureLog(log)
	var patient MedicineInfoPatientRow
	err := pdb.db.Raw(query, prescriptionID).Scan(&patient).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "GetPatientByPrescriptionID"), zap.Error(err))
		return MedicineInfoPatientRow{}, err
	}
	return patient, nil
}

func (pdb *PrescriptionDB) GetQtyInfoByMed(log *zap.Logger, prescriptionID string) ([]PrescriptionItems, error) {
	log = ensureLog(log)
	var prescriptionItems []PrescriptionItems
	err := pdb.db.Model(&PrescriptionItems{}).Where("prescription_id=?", prescriptionID).Select("id", "medicine_id", "quantity", "balance_after_dispense").Find(&prescriptionItems).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "GetQtyInfoByMed"), zap.Error(err))
		return nil, err
	}
	return prescriptionItems, nil
}

func (pdb *PrescriptionDB) GetItemStatusesByPrescriptionID(log *zap.Logger, tx *gorm.DB, prescriptionID string) ([]PrescriptionItems, error) {
	log = ensureLog(log)
	db := pdb.db
	if tx != nil {
		db = tx
	}
	var items []PrescriptionItems
	err := db.Model(&PrescriptionItems{}).
		Where("prescription_id = ?", prescriptionID).
		Select("id", "status", "quantity", "balance_after_dispense").
		Find(&items).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "GetItemStatusesByPrescriptionID"), zap.Error(err))
		return nil, err
	}
	return items, err
}

func (pdb *PrescriptionDB) UpdateDispenseItemQty(log *zap.Logger, tx *gorm.DB, query string, prescriptionItemID string, dispensedQty int64) error {
	log = ensureLog(log)
	err := tx.Exec(query, dispensedQty, prescriptionItemID).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "UpdateDispenseItemQty"), zap.Error(err))
		return err
	}
	return nil
}

func (pdb *PrescriptionDB) UpdatePrescriptionItemStatus(log *zap.Logger, tx *gorm.DB, prescriptionItemID string, status string, outOfStock bool) error {
	log = ensureLog(log)
	// map so out_of_stock=false is persisted (GORM Updates skips zero-value bools on structs)
	err := tx.Model(&PrescriptionItems{}).Where("id = ?", prescriptionItemID).Updates(map[string]interface{}{
		"status":       status,
		"out_of_stock": outOfStock,
		"updated_at":   time.Now(),
	}).Error
	if err != nil {
		log.Error("prescription item repo error", zap.String("op", "UpdatePrescriptionItemStatus"), zap.Error(err))
		return err
	}
	return nil
}
