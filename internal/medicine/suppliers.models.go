package medicine

import "time"

type Paymentterms string
type SupplierStatus string
type SupplierCode string

var (
	Net30    Paymentterms   = "Net 30"
	Net15    Paymentterms   = "Net 15"
	Net45    Paymentterms   = "Net 45"
	Advance  Paymentterms   = "Advance"
	ICash    Paymentterms   = "Cash"
	Active   SupplierStatus = "Active"
	Inactive SupplierStatus = "Inactive"
	SUPP     SupplierCode   = "SUPP"
)

type Supplier struct {
	ID             string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SupplierCode   string         `json:"supplier_code" gorm:"type:varchar(50);not null;uniqueIndex:idx_org_supplier_code"`
	Name           string         `json:"name" gorm:"type:varchar(255);not null"`
	ContactNumber  string         `json:"contact_number" gorm:"type:varchar(20);not null"`
	Email          string         `json:"email" gorm:"type:varchar(255);not null"`
	Address        string         `json:"address" gorm:"type:varchar(500);not null"`
	OrganisationID string         `json:"organisation_id" gorm:"type:uuid;not null;uniqueIndex:idx_org_supplier_code"`
	GstNumber      string         `json:"gst_number" gorm:"type:varchar(50)"`
	DrugLicenseNo  string         `json:"drug_license_no" gorm:"type:varchar(50)"`
	PaymentTerms   Paymentterms   `json:"payment_terms" gorm:"type:varchar(255)"`
	SupplierStatus SupplierStatus `json:"supplier_status" gorm:"type:varchar(255)"`
	CreditLimit    float64        `json:"credit_limit" gorm:"type:numeric(10,2)"`
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedBy      string         `json:"created_by" gorm:"type:uuid;not null"`
}
