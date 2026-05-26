package medicine

import (
	"time"

	"gorm.io/gorm"
)

type PurchaseEntryService struct {
	VPurchaseEntry RPurchaseEntry
}

func NewPurchaseEntryService(PurchaseEntryRepo RPurchaseEntry) *PurchaseEntryService {
	return &PurchaseEntryService{VPurchaseEntry: PurchaseEntryRepo}
}

func (PService *PurchaseEntryService) CreatePurchaseEntry(db *gorm.DB, purchaseEntry *MPurchaseEntry, paymentTerms Paymentterms) error {
	PService.calculatePaymentDueDate(purchaseEntry.InvoiceDate, paymentTerms)
	return PService.VPurchaseEntry.CreatePurchaseEntry(db, purchaseEntry)
}
func (Pservice *PurchaseEntryService) calculatePaymentDueDate(invoiceDate time.Time, paymentTerms Paymentterms) time.Time {
	// Parse the invoice date
	// layout := "2006-01-02" // Assuming the date is in YYYY-MM-DD format
	switch paymentTerms {
	case ICash:
		return invoiceDate
	case Net30:
		return invoiceDate.AddDate(0, 0, 30)
	case Net15:
		return invoiceDate.AddDate(0, 0, 15)
	case Advance:
		return invoiceDate
	case Net45:
		return invoiceDate.AddDate(0, 0, 45)
	default:
		return invoiceDate.AddDate(0, 0, 30)
	}

}
