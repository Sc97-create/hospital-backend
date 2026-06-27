package medicine

import (
	"fmt"
	"hospital-backend/internal/medicine/dto"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MedicineService struct {
	Db       *gorm.DB
	Mrepo    MedicineRepository
	SMed     SMedicineInventory
	SMvmt    SMedicineMvmt
	PurEntry PurchaseEntryService
	Supplier SupplierService
}

func NewMedicineService(db *gorm.DB, Mrepo MedicineRepository, SMed SMedicineInventory, SMvmt SMedicineMvmt, PurEntry PurchaseEntryService, Supplier SupplierService) *MedicineService {
	return &MedicineService{Db: db, Mrepo: Mrepo, SMed: SMed, SMvmt: SMvmt, PurEntry: PurEntry, Supplier: Supplier}
}

func (MService *MedicineService) CreateMedicine(MedicinePayload dto.RequestPayload) error {
	MedicinePayload.InvoiceDate = time.Now()
	purchaseEntry := MService.toPurchaseEntry(MedicinePayload)
	medicines := MService.toMedicine(MedicinePayload.MedicineArray, MedicinePayload.UserID, MedicinePayload.OrganisationID)
	medicineInventory := MService.toMedicineInventory(MedicinePayload.MedicineArray, MedicinePayload.UserID, MedicinePayload.OrganisationID, MedicinePayload.SupplierID, purchaseEntry.ID)
	medicineMvmt := MService.toMedicineMvmt(MedicinePayload.MedicineArray, MedicinePayload.UserID, MedicinePayload.OrganisationID)

	//get supplier details by supplier id
	supplier, err := MService.Supplier.GetSupplierByID(MedicinePayload.SupplierID)
	if err != nil {
		return err
	}
	tx := MService.Db.Begin()
	err = MService.PurEntry.CreatePurchaseEntry(tx, purchaseEntry, supplier.PaymentTerms)
	if err != nil {
		tx.Rollback()
		return err
	}
	err = MService.Mrepo.CreateInBatches(tx, medicines)
	if err != nil {
		tx.Rollback()
		return err
	}
	err = MService.SMed.CreateMedicineInventory(tx, medicineInventory)
	if err != nil {
		tx.Rollback()
		return err
	}
	err = MService.SMvmt.CreateMedicineMvmt(tx, medicineMvmt)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (Mservice *MedicineService) toPurchaseEntry(payload dto.RequestPayload) *MPurchaseEntry {
	var purchaseEntry MPurchaseEntry
	purchaseEntry.ID = uuid.NewString()
	purchaseEntry.InvoiceNumber = payload.InvoiceNo
	purchaseEntry.InvoiceDate = payload.InvoiceDate
	purchaseEntry.SupplierID = payload.SupplierID
	purchaseEntry.OrganisationID = payload.OrganisationID
	if payload.PaymentDueDate == "" {
		purchaseEntry.PaymentDueDate, _ = time.Parse(time.RFC3339, payload.PaymentDueDate)
	}
	purchaseEntry.PaymentDueDate = time.Now()
	return &purchaseEntry
}
func (Mservice *MedicineService) toMedicineMvmt(meds []dto.MedicineInfo, userID string, organisationID string) []MedicineStockMovements {
	var MedMvmt []MedicineStockMovements
	for _, each := range meds {
		var medMvmt MedicineStockMovements
		medMvmt.ID = uuid.NewString()
		medMvmt.MedicineID = each.MedicineID
		medMvmt.MedicineInventoryID = each.MedInventoryID
		medMvmt.OrganisationID = organisationID
		medMvmt.MovementType = Purchase
		medMvmt.QtyChanged = each.PurchaseQtyBoxes * each.UnitPerBoxes
		medMvmt.CreatedBy = userID
		medMvmt.SourceType = PurchaseEntry
		medMvmt.UnitPriceAtTimeOfMvmt = each.SellingPrice / float64(each.UnitPerBoxes)
		MedMvmt = append(MedMvmt, medMvmt)
	}
	return MedMvmt
}
func (Mservice *MedicineService) toMedicine(med []dto.MedicineInfo, userID string, organisationID string) []Medicine {
	var Medicines []Medicine
	for _, each := range med {
		if !each.Add {
			continue
		}
		var medicine Medicine
		medicine.ID = each.MedicineID
		medicine.Code = Mservice.createCode(MedicineCodePrefix)
		medicine.Name = each.Name
		medicine.Form = each.Form
		medicine.Strength = each.Strength
		medicine.CreatedAt = time.Now()
		medicine.CreatedBy = userID
		medicine.OrganisationID = organisationID
		medicine.HSNCode = each.HsnCode
		medicine.ReorderLevel = each.ReorderLevel
		medicine.MaxStockTarget = each.MaxStockTarget
		Medicines = append(Medicines, medicine)
	}
	return Medicines
}
func (Mservice *MedicineService) createCode(prefix CodePrefix) string {
	return fmt.Sprintf("%s-%d", prefix, rand.Intn(9000)+1000)
}
func (Mservice *MedicineService) toMedicineInventory(med []dto.MedicineInfo, userID string, organisationID string, supplierID string, purchaseEntryID string) []MedicineInventory {
	var MedInventorys []MedicineInventory

	for _, each := range med {
		var MedInventory MedicineInventory
		MedInventory.ID = each.MedInventoryID
		MedInventory.MedicineID = each.MedicineID
		MedInventory.BatchNo = each.BatchNumber

		MedInventory.SupplierID = supplierID
		MedInventory.OrganisationID = organisationID
		MedInventory.PurchaseEntryID = purchaseEntryID
		MedInventory.PurchaseQtyBoxes = each.PurchaseQtyBoxes
		MedInventory.UnitsPerBox = each.UnitPerBoxes
		MedInventory.ExpiresAt, _ = time.Parse(time.RFC3339, each.ExpiryDate)
		MedInventory.ShelfLocation = each.ShelfLocation
		MedInventory.CurrentStockUnits = each.PurchaseQtyBoxes * each.UnitPerBoxes
		MedInventory.CreatedAt = time.Now()
		MedInventory.CreatedBy = userID
		MedInventory.Pricing.PurchasePrice = each.PurchasePrice
		if each.SellingPrice == 0 {
			MedInventory.Pricing.SellingPrice = each.MRP
		}
		MedInventory.Pricing.SellingPrice = each.SellingPrice
		MedInventory.Pricing.MRP = each.MRP
		MedInventory.Pricing.Discount = each.Discount
		MedInventory.Pricing.DiscountType = "amount"
		MedInventory.Pricing.UnitPrice = each.MRP / float64(each.UnitPerBoxes)
		MedInventory.Pricing.TotalPrice = (each.PurchasePrice - each.Discount) * float64(each.PurchaseQtyBoxes)
		MedInventorys = append(MedInventorys, MedInventory)
	}
	return MedInventorys

}
func (Mservice *MedicineService) GetOne(id string) (*Medicine, error) {
	med, err := Mservice.Mrepo.FindOne(id)
	if err != nil {
		return nil, err
	}
	return med, nil
}
func (Mservice *MedicineService) GetMany(limit int, pageno int) (Med []Medicine, err error) {
	skip := 0
	if pageno != 0 {
		skip = (pageno - 1) * limit
	}
	Med, err = Mservice.GetMany(limit, skip)
	if err != nil {
		return
	}
	return
}
func (Mservice *MedicineService) SearchMedicine(name string) ([]Medicine, error) {
	name = strings.TrimSpace(name)

	query := `
	($1 = ''
	OR name ILIKE $2
	OR code ILIKE $2)
`

	// prefix match: starts with the provided name (case-insensitive)
	pattern := name + "%"

	args := []interface{}{
		name,
		pattern,
	}
	return Mservice.Mrepo.FindMany(query, args...)
}
func (Mservice *MedicineService) FindNamesByIds(ids []string) (Med []Medicine, err error) {
	Med, err = Mservice.Mrepo.FindNamesByIds(ids)
	if err != nil {
		return
	}

	return
}
func (Mservice *MedicineService) toMedicineResponse(Med []Medicine) []dto.MedicineResponse {
	medicineResponse := []dto.MedicineResponse{}
	for _, each := range Med {
		medicineResponse = append(medicineResponse, dto.MedicineResponse{
			ID:   each.ID,
			Name: each.Name,
		})
	}
	return medicineResponse
}

/*
if pharma loads medicine then in medicine movement
		  purchase entry table stores record with supplierid, invoice no, invoice date
		  purchase entry items table stores each entry from the table
	      for every purchase entry item if medicine is present then too we have to store it in new record in medicine inventory with batch number and expiry date and qty added to inventory
		  store one record for that medicine in movement with purchase type and quantity added, unit price at time of movement and balance after movement
		  if there is order from patient then from medicine inventory and update the available qty
		  store one record for that medicine in movement with patient order item  type and quantity reduced.
*/
