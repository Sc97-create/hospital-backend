package medicine

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ISupplier interface {
	CretateSupplier(log *zap.Logger, supplier *Supplier) error
	GetSupplierByID(log *zap.Logger, supplierID string) (Supplier, error)
	GetSupplierByOrgID(log *zap.Logger, query string, args ...any) ([]Supplier, error)
	CountSupplierByOrgID(log *zap.Logger, query string, args ...any) (int64, error)
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

func (Srepo *MedicineRepo) GetSupplierByOrgID(log *zap.Logger, query string, args ...any) ([]Supplier, error) {
	log = ensureLog(log)
	var suppliers []Supplier
	err := Srepo.db.Raw(query, args...).Scan(&suppliers).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "GetSupplierByOrgID"), zap.Error(err))
		return nil, err
	}
	if suppliers == nil {
		suppliers = []Supplier{}
	}
	return suppliers, nil
}

func (Srepo *MedicineRepo) CountSupplierByOrgID(log *zap.Logger, query string, args ...any) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := Srepo.db.Raw(query, args...).Scan(&count).Error
	if err != nil {
		log.Error("medicine repo error", zap.String("op", "CountSupplierByOrgID"), zap.Error(err))
		return 0, err
	}
	return count, nil
}
