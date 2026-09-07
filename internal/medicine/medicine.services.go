package medicine

import (
	"errors"
	"fmt"
	"hospital-backend/internal/medicine/dto"
	"hospital-backend/pkg/types"
	wrapError "hospital-backend/shared/error"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MedicineService struct {
	Db       *gorm.DB
	Mrepo    MedicineRepository
	SMed     InventoryCreator
	SMvmt    MvmtCreator
	PurEntry PurchaseEntryCreator
	Supplier SupplierLookup
}

type InventoryCreator interface {
	CreateMedicineInventory(log *zap.Logger, db *gorm.DB, medicineInventory []MedicineInventory) error
}

type MvmtCreator interface {
	CreateMedicineMvmt(log *zap.Logger, db *gorm.DB, medicineMvmt []types.MedicineStockMovements) error
}

type PurchaseEntryCreator interface {
	CreatePurchaseEntry(log *zap.Logger, db *gorm.DB, purchaseEntry *MPurchaseEntry, paymentTerms Paymentterms) error
}

type SupplierLookup interface {
	GetSupplierByID(log *zap.Logger, supplierID string) (Supplier, error)
}

func NewMedicineService(db *gorm.DB, Mrepo MedicineRepository, SMed InventoryCreator, SMvmt MvmtCreator, PurEntry PurchaseEntryCreator, Supplier SupplierLookup) *MedicineService {
	return &MedicineService{Db: db, Mrepo: Mrepo, SMed: SMed, SMvmt: SMvmt, PurEntry: PurEntry, Supplier: Supplier}
}

func (MService *MedicineService) CreateMedicine(log *zap.Logger, MedicinePayload dto.RequestPayload) error {
	log = ensureLog(log)
	MedicinePayload.InvoiceDate = time.Now()
	purchaseEntry := MService.toPurchaseEntry(MedicinePayload)
	medicines := MService.toMedicine(MedicinePayload.MedicineArray, MedicinePayload.UserID, MedicinePayload.OrganisationID)
	medicineInventory := MService.toMedicineInventory(MedicinePayload.MedicineArray, MedicinePayload.UserID, MedicinePayload.OrganisationID, MedicinePayload.SupplierID, purchaseEntry.ID)
	medicineMvmt := MService.toMedicineMvmt(MedicinePayload.MedicineArray, MedicinePayload.UserID, MedicinePayload.OrganisationID)

	supplier, err := MService.Supplier.GetSupplierByID(log, MedicinePayload.SupplierID)
	if err != nil {
		if errors.Is(err, wrapError.ErrSupplierNotFound) {
			log.Warn("medicine purchase failed",
				zap.String("supplier_id", MedicinePayload.SupplierID),
				zap.String("organisation_id", MedicinePayload.OrganisationID),
				zap.String("reason", "supplier_not_found"),
			)
			return wrapError.ErrSupplierNotFound
		}
		log.Error("medicine purchase failed",
			zap.String("supplier_id", MedicinePayload.SupplierID),
			zap.String("organisation_id", MedicinePayload.OrganisationID),
			zap.String("reason", "supplier_lookup"),
			zap.Error(err),
		)
		return wrapError.ErrMedicinePurchaseFailed
	}

	tx := MService.Db.Begin()
	err = MService.PurEntry.CreatePurchaseEntry(log, tx, purchaseEntry, supplier.PaymentTerms)
	if err != nil {
		tx.Rollback()
		log.Error("medicine purchase failed",
			zap.String("organisation_id", MedicinePayload.OrganisationID),
			zap.String("supplier_id", MedicinePayload.SupplierID),
			zap.String("reason", "purchase_entry"),
			zap.Error(err),
		)
		return wrapError.ErrMedicinePurchaseFailed
	}
	err = MService.Mrepo.CreateInBatches(log, tx, medicines)
	if err != nil {
		tx.Rollback()
		log.Error("medicine purchase failed",
			zap.String("organisation_id", MedicinePayload.OrganisationID),
			zap.String("reason", "medicines_create"),
			zap.Error(err),
		)
		return wrapError.ErrMedicinePurchaseFailed
	}
	err = MService.SMed.CreateMedicineInventory(log, tx, medicineInventory)
	if err != nil {
		tx.Rollback()
		log.Error("medicine purchase failed",
			zap.String("organisation_id", MedicinePayload.OrganisationID),
			zap.String("purchase_entry_id", purchaseEntry.ID),
			zap.String("reason", "inventory_create"),
			zap.Error(err),
		)
		return wrapError.ErrMedicinePurchaseFailed
	}
	err = MService.SMvmt.CreateMedicineMvmt(log, tx, medicineMvmt)
	if err != nil {
		tx.Rollback()
		log.Error("medicine purchase failed",
			zap.String("organisation_id", MedicinePayload.OrganisationID),
			zap.String("reason", "stock_mvmt"),
			zap.Error(err),
		)
		return wrapError.ErrMedicinePurchaseFailed
	}
	if err = tx.Commit().Error; err != nil {
		log.Error("medicine purchase failed",
			zap.String("organisation_id", MedicinePayload.OrganisationID),
			zap.String("reason", "db_commit"),
			zap.Error(err),
		)
		return wrapError.ErrMedicinePurchaseFailed
	}

	log.Info("medicine purchase success",
		zap.String("organisation_id", MedicinePayload.OrganisationID),
		zap.String("supplier_id", MedicinePayload.SupplierID),
		zap.String("purchase_entry_id", purchaseEntry.ID),
		zap.String("invoice_no", MedicinePayload.InvoiceNo),
		zap.Int("item_count", len(MedicinePayload.MedicineArray)),
		zap.Int("new_medicine_count", len(medicines)),
		zap.Int("inventory_count", len(medicineInventory)),
		zap.Int("movement_count", len(medicineMvmt)),
	)
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

func (Mservice *MedicineService) toMedicineMvmt(meds []dto.MedicineInfo, userID string, organisationID string) []types.MedicineStockMovements {
	var MedMvmt []types.MedicineStockMovements
	for _, each := range meds {
		var medMvmt types.MedicineStockMovements
		medMvmt.ID = uuid.NewString()
		medMvmt.MedicineID = each.MedicineID
		medMvmt.MedicineInventoryID = each.MedInventoryID
		medMvmt.OrganisationID = organisationID
		medMvmt.MovementType = types.Purchase
		medMvmt.QtyChanged = each.PurchaseQtyBoxes * each.UnitPerBoxes
		medMvmt.CreatedBy = userID
		medMvmt.SourceType = types.PurchaseEntry
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

func (Mservice *MedicineService) GetOne(log *zap.Logger, id string) (*Medicine, error) {
	log = ensureLog(log)
	med, err := Mservice.Mrepo.FindOne(log, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, wrapError.ErrMedicineNotFound
		}
		return nil, err
	}
	return med, nil
}

func (Mservice *MedicineService) GetMany(log *zap.Logger, limit int, pageno int) (Med []Medicine, err error) {
	log = ensureLog(log)
	skip := 0
	if pageno != 0 {
		skip = (pageno - 1) * limit
	}
	Med, err = Mservice.Mrepo.FindMany(log, "1=1 LIMIT ? OFFSET ?", limit, skip)
	if err != nil {
		return
	}
	return
}

func (Mservice *MedicineService) SearchMedicine(log *zap.Logger, name string, organisationID string) ([]dto.SearchMedicineItem, error) {
	log = ensureLog(log)
	name = strings.TrimSpace(name)
	organisationID = strings.TrimSpace(organisationID)
	pattern := name + "%"
	results, err := Mservice.Mrepo.SearchMedicine(log, name, pattern, organisationID)
	if err != nil {
		log.Error("medicine search failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, wrapError.ErrMedicineSearchFailed
	}
	log.Info("medicine search success",
		zap.String("organisation_id", organisationID),
		zap.Int("result_count", len(results)),
	)
	return results, nil
}

func (Mservice *MedicineService) FindNamesByIds(log *zap.Logger, ids []string) (Med []Medicine, err error) {
	return Mservice.Mrepo.FindNamesByIds(ensureLog(log), ids)
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
