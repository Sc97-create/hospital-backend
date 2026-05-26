package medcontainer

import (
	"hospital-backend/internal/medicine"

	"gorm.io/gorm"
)

type MedContainer struct {
	Medicineservices     *medicine.MedicineService
	MedInventoryService  *medicine.SMedicineInventory
	MedMvmtService       *medicine.SMedicineMvmt
	PurchaseEntryService *medicine.PurchaseEntryService
	SupplierService      *medicine.SupplierService
}

func MedicineContainer(db *gorm.DB) *MedContainer {
	medicineRepo := medicine.NewMedicineRepo(db)
	medInventoryRepo := medicine.NewMedicineRepo(db)
	medMvmtRepo := medicine.NewMedicineRepo(db)
	purchaseEntryRepo := medicine.NewMedicineRepo(db)
	supplierRepo := medicine.NewMedicineRepo(db)
	//medicineService := medicine.NewMedicineService()
	medInventoryService := medicine.NewSMedicineInventory(medInventoryRepo)
	medMvmtService := medicine.NewMedicineMvmt(medMvmtRepo)
	purchaseEntryService := medicine.NewPurchaseEntryService(purchaseEntryRepo)
	supplierService := medicine.NewSupplierService(supplierRepo)
	medicineService := medicine.NewMedicineService(db, medicineRepo, *medInventoryService, *medMvmtService, *purchaseEntryService, *supplierService)

	return &MedContainer{
		Medicineservices:     medicineService,
		MedInventoryService:  medInventoryService,
		MedMvmtService:       medMvmtService,
		PurchaseEntryService: purchaseEntryService,
		SupplierService:      supplierService,
	}
}
