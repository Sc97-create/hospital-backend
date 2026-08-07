package medicine

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ISupplier interface {
	CretateSupplier(log *zap.Logger, supplier *Supplier) error
	GetSupplierByID(log *zap.Logger, supplierID string) (Supplier, error)
	GetSupplierByOrgID(log *zap.Logger, organisationID string, limit int, offset int) ([]Supplier, error)
	CountSupplierByOrgID(log *zap.Logger, organisationID string) (int64, error)
}

func (Srepo *MedicineRepo) CretateSupplier(log *zap.Logger, supplier *Supplier) error {
	log = ensureLog(log)
	err := Srepo.db.Create(&supplier).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "CretateSupplier"), zap.Error(err))
		return err
	}
	return nil
}

func (Srepo *MedicineRepo) GetSupplierByID(log *zap.Logger, supplierID string) (Supplier, error) {
	log = ensureLog(log)
	var supplier Supplier
	err := Srepo.db.Model(Supplier{}).Where("id=?", supplierID).Select("id,name,payment_terms").First(&supplier).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("medicine repo error", zap.String("op", "GetSupplierByID"), zap.Error(err))
		}
		return Supplier{}, err
	}
	return supplier, nil
}

func (Srepo *MedicineRepo) GetSupplierByOrgID(log *zap.Logger, organisationID string, limit int, offset int) ([]Supplier, error) {
	log = ensureLog(log)
	var suppliers []Supplier
	err := Srepo.db.Model(&Supplier{}).
		Select("id, supplier_code, name, contact_number, email, payment_terms, supplier_status, created_at").
		Where("organisation_id = ?", organisationID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&suppliers).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "GetSupplierByOrgID"), zap.Error(err))
		return nil, err
	}
	if suppliers == nil {
		suppliers = []Supplier{}
	}
	return suppliers, nil
}

func (Srepo *MedicineRepo) CountSupplierByOrgID(log *zap.Logger, organisationID string) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := Srepo.db.Model(&Supplier{}).Where("organisation_id = ?", organisationID).Count(&count).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "CountSupplierByOrgID"), zap.Error(err))
		return 0, err
	}
	return count, nil
}
