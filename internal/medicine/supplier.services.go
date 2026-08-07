package medicine

import (
	"errors"
	"fmt"
	"hospital-backend/internal/medicine/dto"
	"hospital-backend/shared/commonfunctions"
	wrapError "hospital-backend/shared/error"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SupplierService struct {
	SupplierRepo ISupplier
}

func NewSupplierService(SupplierRepo ISupplier) *SupplierService {
	return &SupplierService{SupplierRepo: SupplierRepo}
}

func (SService *SupplierService) CretateSupplier(log *zap.Logger, supplier dto.Supplier) (string, error) {
	log = ensureLog(log)
	supplierData := SService.toSupplier(supplier)
	err := SService.SupplierRepo.CretateSupplier(log, &supplierData)
	if err != nil {
		log.Error("supplier create failed",
			zap.String("organisation_id", supplier.OrganisationID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", wrapError.ErrSupplierCreateFailed
	}
	log.Info("supplier create success",
		zap.String("organisation_id", supplier.OrganisationID),
		zap.String("supplier_id", supplierData.ID),
		zap.String("supplier_code", supplierData.SupplierCode),
		zap.String("payment_terms", string(supplierData.PaymentTerms)),
	)
	return supplierData.ID, nil
}

func (SService *SupplierService) GetSupplierByID(log *zap.Logger, supplierID string) (Supplier, error) {
	log = ensureLog(log)
	supplier, err := SService.SupplierRepo.GetSupplierByID(log, supplierID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("supplier get failed",
				zap.String("supplier_id", supplierID),
				zap.String("reason", "not_found"),
			)
			return Supplier{}, wrapError.ErrSupplierNotFound
		}
		log.Error("supplier get failed",
			zap.String("supplier_id", supplierID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return Supplier{}, wrapError.ErrSupplierFetchFailed
	}
	log.Debug("supplier get success", zap.String("supplier_id", supplier.ID))
	return supplier, nil
}

func (SService *SupplierService) GetSupplierByOrgID(log *zap.Logger, organisationID string, limit int, pageNo int) ([]dto.SupplierListItem, int64, error) {
	log = ensureLog(log)
	if limit <= 0 {
		limit = 10
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	offset := commonfunctions.Getskip(limit, pageNo)
	suppliers, err := SService.SupplierRepo.GetSupplierByOrgID(log, organisationID, limit, offset)
	if err != nil {
		log.Error("supplier list failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrSupplierFetchFailed
	}
	total, err := SService.GetTotalCount(log, organisationID)
	if err != nil {
		return nil, 0, err
	}
	log.Info("supplier list success",
		zap.String("organisation_id", organisationID),
		zap.Int("result_count", len(suppliers)),
		zap.Int("limit", limit),
		zap.Int("page_no", pageNo),
	)
	return SService.toSupplierList(suppliers), total, nil
}

func (SService *SupplierService) GetTotalCount(log *zap.Logger, organisationID string) (int64, error) {
	log = ensureLog(log)
	total, err := SService.SupplierRepo.CountSupplierByOrgID(log, organisationID)
	if err != nil {
		log.Error("supplier count failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return 0, wrapError.ErrSupplierFetchFailed
	}
	return total, nil
}

func (SService *SupplierService) toSupplierList(suppliers []Supplier) []dto.SupplierListItem {
	list := make([]dto.SupplierListItem, 0, len(suppliers))
	for _, each := range suppliers {
		list = append(list, dto.SupplierListItem{
			ID:             each.ID,
			SupplierCode:   each.SupplierCode,
			Name:           each.Name,
			ContactNumber:  each.ContactNumber,
			Email:          each.Email,
			PaymentTerms:   string(each.PaymentTerms),
			SupplierStatus: string(each.SupplierStatus),
			CreatedAt:      each.CreatedAt.Format("02 Jan 2006"),
		})
	}
	return list
}

func (SService *SupplierService) toSupplier(supplier dto.Supplier) Supplier {
	return Supplier{
		ID:             uuid.New().String(),
		SupplierStatus: Active,
		SupplierCode:   SService.createCode(SUPP),
		CreatedBy:      supplier.UserID,
		CreatedAt:      time.Now(),
		OrganisationID: supplier.OrganisationID,
		Name:           supplier.Name,
		PaymentTerms:   SService.findPaymentTerms(supplier.PaymentTerms),
		Email:          supplier.EmailID,
		DrugLicenseNo:  supplier.DrugLicenseNumber,
		ContactNumber:  supplier.ContactNumber,
		CreditLimit:    supplier.CreditLimit,
		GstNumber:      supplier.GstNumber,
	}
}

func (SService *SupplierService) createCode(prefix SupplierCode) string {
	return fmt.Sprintf("%s-%d", prefix, rand.Intn(9000)+1000)
}

func (SService *SupplierService) findPaymentTerms(paymentTerms string) Paymentterms {
	switch paymentTerms {
	case "Cash":
		return ICash
	case "Net 30":
		return Net30
	case "Net 45":
		return Net45
	case "Net 15":
		return Net15
	default:
		return Advance
	}
}
