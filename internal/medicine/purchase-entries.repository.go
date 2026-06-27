package medicine

import "gorm.io/gorm"

type RPurchaseEntry interface {
	CreatePurchaseEntry(db *gorm.DB, purchaseEntry *MPurchaseEntry) error
}

func (r *MedicineRepo) CreatePurchaseEntry(db *gorm.DB, purchaseEntry *MPurchaseEntry) error {
	return db.Create(&purchaseEntry).Error
}
