package medicine

import (
	"errors"
	"fmt"
	"hospital-backend/internal/medicine/dto"
	wrapError "hospital-backend/shared/error"
	"math/rand"
	"strings"
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

func (SService *SupplierService) GetSupplierByOrgID(log *zap.Logger, req dto.SupplierListReq) ([]dto.SupplierListItem, int64, error) {
	log = ensureLog(log)
	req.Search = strings.TrimSpace(req.Search)
	req.DBLimit, req.DBOffset = SService.parsePagination(req.Limit, req.PageNo)

	listQuery, listArgs := SService.buildSupplierListQuery(req)
	suppliers, err := SService.SupplierRepo.GetSupplierByOrgID(log, listQuery, listArgs...)
	if err != nil {
		log.Error("supplier list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrSupplierFetchFailed
	}

	countQuery, countArgs := SService.buildSupplierCountQuery(req)
	total, err := SService.SupplierRepo.CountSupplierByOrgID(log, countQuery, countArgs...)
	if err != nil {
		log.Error("supplier list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrSupplierFetchFailed
	}

	log.Info("supplier list success",
		zap.String("organisation_id", req.OrganisationID),
		zap.Int("result_count", len(suppliers)),
		zap.Int64("total", total),
		zap.Int("limit", req.DBLimit),
		zap.Bool("has_search", req.Search != ""),
	)
	return SService.toSupplierList(suppliers), total, nil
}

func (SService *SupplierService) GetTotalCount(log *zap.Logger, organisationID string) (int64, error) {
	log = ensureLog(log)
	query := `SELECT COUNT(*) FROM suppliers WHERE organisation_id = $1`
	total, err := SService.SupplierRepo.CountSupplierByOrgID(log, query, organisationID)
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

func (SService *SupplierService) buildSupplierListQuery(req dto.SupplierListReq) (string, []interface{}) {
	baseQuery := `
		SELECT
			id,
			supplier_code,
			name,
			contact_number,
			email,
			payment_terms,
			supplier_status,
			created_at
		FROM suppliers
		WHERE organisation_id = $1
	`
	args := []interface{}{req.OrganisationID}
	baseQuery, args, argsPos := SService.appendSupplierFilters(baseQuery, req, args, 2)
	baseQuery += " ORDER BY created_at DESC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argsPos, argsPos+1)
	args = append(args, req.DBLimit, req.DBOffset)
	return baseQuery, args
}

func (SService *SupplierService) buildSupplierCountQuery(req dto.SupplierListReq) (string, []interface{}) {
	countQuery := `SELECT COUNT(*) FROM suppliers WHERE organisation_id = $1`
	args := []interface{}{req.OrganisationID}
	countQuery, args, _ = SService.appendSupplierFilters(countQuery, req, args, 2)
	return countQuery, args
}

func (SService *SupplierService) appendSupplierFilters(query string, req dto.SupplierListReq, args []interface{}, argsPos int) (string, []interface{}, int) {
	if req.Search == "" {
		return query, args, argsPos
	}
	query += fmt.Sprintf(" AND (name ILIKE $%d OR supplier_code ILIKE $%d)", argsPos, argsPos)
	args = append(args, "%"+req.Search+"%")
	argsPos++
	return query, args, argsPos
}

func (SService *SupplierService) parsePagination(limit float64, pageNo float64) (int, int) {
	numLimit := int(limit)
	if numLimit <= 0 {
		numLimit = 10
	}
	numPage := int(pageNo)
	if numPage <= 0 {
		numPage = 1
	}
	return numLimit, (numPage - 1) * numLimit
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
