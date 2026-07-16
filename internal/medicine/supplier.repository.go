package medicine

type ISupplier interface {
	CretateSupplier(supplier *Supplier) error
	GetSupplierByID(supplierID string) (Supplier, error)
	GetSupplierByOrgID(organisationID string, limit int, offset int) ([]Supplier, error)
	CountSupplierByOrgID(organisationID string) (int64, error)
}

func (Srepo *MedicineRepo) CretateSupplier(supplier *Supplier) error {
	return Srepo.db.Create(&supplier).Error
}
func (Srepo *MedicineRepo) GetSupplierByID(supplierID string) (Supplier, error) {
	var supplier Supplier
	err := Srepo.db.Model(Supplier{}).Where("id=?", supplierID).Select("id,name,payment_terms").First(&supplier).Error
	if err != nil {
		return Supplier{}, err
	}
	return supplier, err
}
func (Srepo *MedicineRepo) GetSupplierByOrgID(organisationID string, limit int, offset int) ([]Supplier, error) {
	var suppliers []Supplier
	err := Srepo.db.Model(&Supplier{}).
		Select("id, supplier_code, name, contact_number, email, payment_terms, supplier_status, created_at").
		Where("organisation_id = ?", organisationID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&suppliers).Error
	if err != nil {
		return nil, err
	}
	if suppliers == nil {
		suppliers = []Supplier{}
	}
	return suppliers, nil
}
func (Srepo *MedicineRepo) CountSupplierByOrgID(organisationID string) (int64, error) {
	var count int64
	err := Srepo.db.Model(&Supplier{}).Where("organisation_id = ?", organisationID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
