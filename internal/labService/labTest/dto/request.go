package dto

// CreateLabOrderRequest is the body for POST /lab-orders/create.
type CreateLabOrderRequest struct {
	OrganisationID     string                   `json:"organisation_id"`
	PatientID          string                   `json:"patient_id"`
	DoctorID           string                   `json:"doctor_id"`
	DiagnosticCentreID string                   `json:"diagnostic_centre_id"`
	AppointmentID      string                   `json:"appointment_id"`
	Tests              []CreateOrderedTestInput `json:"tests"`
}

// CreateOrderedTestInput is one test line on create.
type CreateOrderedTestInput struct {
	Name    string `json:"name"`
	SubTest string `json:"sub_test"`
}

// ListLabOrdersRequest is query input for GET /lab-orders/list.
type ListLabOrdersRequest struct {
	OrganisationID string  `json:"organisation_id" query:"organisation_id"`
	PatientID      string  `json:"patient_id" query:"patient_id"`
	Status         string  `json:"status" query:"status"`
	Limit          float64 `json:"limit" query:"limit"`
	PageNo         float64 `json:"page_no" query:"page_no"`
	DBLimit        int     `json:"-"`
	DBOffset       int     `json:"-"`
}

// UpdateLabOrderStatusRequest is the body for PATCH .../status.
type UpdateLabOrderStatusRequest struct {
	Status string `json:"status"`
}

// UpdateOrderedTestStatusRequest is the body for PATCH .../tests/:testId/status.
type UpdateOrderedTestStatusRequest struct {
	Status string `json:"status"`
}

// UpdateOrderedTestsStatusRequest is the body for PATCH /lab-orders/bulkupdatestatus.
type UpdateOrderedTestsStatusRequest struct {
	LabOrderID string   `json:"lab_order_id"`
	TestIDs    []string `json:"test_ids"`
	Status     string   `json:"status"`
}

// OrderedTestResponse is one test inside a lab order.
type OrderedTestResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	SubTest string `json:"sub_test"`
	Status  string `json:"status"`
}

// LabOrderListItem is the API shape for GET /lab-orders/list.
type LabOrderListItem struct {
	ID                 string  `json:"id"`
	OrganisationID     string  `json:"organisation_id"`
	PatientID          string  `json:"patient_id"`
	PatientName        string  `json:"patient_name"`
	DoctorID           string  `json:"doctor_id"`
	DoctorName         string  `json:"doctor_name"`
	DiagnosticCentreID string  `json:"diagnostic_centre_id"`
	AppointmentID      *string `json:"appointment_id"`
	TestsCount         int     `json:"tests_count"`
	Status             string  `json:"status"`
	IsPayment          bool    `json:"is_payment"`
	ReceivedAt         string  `json:"received_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// LabOrderPatientDetails is nested patient info on lab-order getbyID.
type LabOrderPatientDetails struct {
	PatientID   string `json:"patient_id"`
	PatientName string `json:"patient_name"`
	PatientCode string `json:"patient_code"`
	Age         int    `json:"age"`
}

// LabOrderDoctorDetails is nested doctor info on lab-order getbyID.
type LabOrderDoctorDetails struct {
	DoctorID   string `json:"doctor_id"`
	DoctorName string `json:"doctor_name"`
}

// LabOrderResponse is the API shape for GET /lab-orders/getbyID.
type LabOrderResponse struct {
	ID                 string                 `json:"id"`
	OrganisationID     string                 `json:"organisation_id"`
	Patient            LabOrderPatientDetails `json:"patient"`
	Doctor             LabOrderDoctorDetails  `json:"doctor"`
	DiagnosticCentreID string                 `json:"diagnostic_centre_id"`
	AppointmentID      *string                `json:"appointment_id"`
	Tests              []OrderedTestResponse  `json:"tests"`
	TestsCount         int                    `json:"tests_count"`
	Status             string                 `json:"status"`
	IsPayment          bool                   `json:"is_payment"`
	ReceivedAt         string                 `json:"received_at"`
	UpdatedAt          string                 `json:"updated_at"`
}
