package dto

type MedicineResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SearchMedicineItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Form           string `json:"form"`
	Strength       string `json:"strength"`
	HsnCode        string `json:"hsn_code"`
	ShelfLocation  string `json:"shelf_location"`
	ReorderLevel   int    `json:"reorder_level"`
	MaxStockTarget int    `json:"max_stock_target"`
}

type SupplierListItem struct {
	ID             string `json:"id"`
	SupplierCode   string `json:"supplier_code"`
	Name           string `json:"name"`
	ContactNumber  string `json:"contact_number"`
	Email          string `json:"email"`
	PaymentTerms   string `json:"payment_terms"`
	SupplierStatus string `json:"supplier_status"`
	CreatedAt      string `json:"created_at"`
}
