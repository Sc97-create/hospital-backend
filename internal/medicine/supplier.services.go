package medicine

import (
	"fmt"
	"hospital-backend/internal/medicine/dto"
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
