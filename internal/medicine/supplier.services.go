package medicine

import (
	"fmt"
	"hospital-backend/internal/medicine/dto"
	"hospital-backend/shared/commonfunctions"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type SupplierService struct {
	SupplierRepo ISupplier
}

func NewSupplierService(SupplierRepo ISupplier) *SupplierService {
	return &SupplierService{SupplierRepo: SupplierRepo}
}
func (SService *SupplierService) CretateSupplier(supplier dto.Supplier) error {
	supplierData := SService.toSupplier(supplier)
	return SService.SupplierRepo.CretateSupplier(&supplierData)
}
func (SService *SupplierService) GetSupplierByID(supplierID string) (Supplier, error) {
	return SService.SupplierRepo.GetSupplierByID(supplierID)
}
func (SService *SupplierService) GetSupplierByOrgID(organisationID string, limit int, pageNo int) ([]dto.SupplierListItem, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	offset := commonfunctions.Getskip(limit, pageNo)
	suppliers, err := SService.SupplierRepo.GetSupplierByOrgID(organisationID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := SService.GetTotalCount(organisationID)
	if err != nil {
		return nil, 0, err
	}
	return SService.toSupplierList(suppliers), total, nil
}
func (SService *SupplierService) GetTotalCount(organisationID string) (int64, error) {
	return SService.SupplierRepo.CountSupplierByOrgID(organisationID)
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
