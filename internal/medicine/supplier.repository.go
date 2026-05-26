package medicine

type ISupplier interface {
	CretateSupplier(supplier *Supplier) error
	GetSupplierByID(supplierID string) (Supplier, error)
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
