package dto

type Request struct {
	Recipient string

	Subject string
	Content string
}
type PrescriptionMedicine struct {
	Name            string `json:"medicine_name"`
	Form            string `json:"medicine_form"`
	Strength        string `json:"medicine_strength"`
	Dosage          string `json:"dosage"`
	Duration        string `json:"duration"`
	Quantity        string `json:"quantity"`
	FoodInstruction string `json:"food_instruction"`
}

type NotificationModel struct {
	PatientName         string                 `json:"patient_name"`
	AppointmentCode     string                 `json:"appointment_code"`
	DoctorName          string                 `json:"doctor_name"`
	HospitalName        string                 `json:"hospital_name"`
	AppointmentDate     string                 `json:"appointment_date"`
	AppointmentTime     string                 `json:"appointment_time"`
	ConsultedOn         string                 `json:"consulted_on"`
	PrescriptionCode    string                 `json:"prescription_code"`
	PrescriptionSummary string                 `json:"prescription_summary"`
	Medicines           []PrescriptionMedicine `json:"medicines"`
	PatientEmail        string                 `json:"patient_email_id"`
	PatientID           string                 `json:"patient_id"`
	EmployeeID          string                 `json:"employee_id"`
	OrganisationID      string                 `json:"organisation_id"`
	PaymentLink         string                 `json:"payment_link"`
	Amount              string                 `json:"amount"`
	Currency            string                 `json:"currency"`
	ExpiresAt           string                 `json:"expires_at"`
	PatientCode         string                 `json:"patient_code"`
	PatientPhone        string                 `json:"patient_phone"`
	PaymentStatus       string                 `json:"payment_status"`
	//Amount          string                 `json:"amount_paid"`
	PaidAt string `json:"paid_at"`

	EmployeeName    string `json:"employee_name"`
	EmployeeEmail   string `json:"employee_email"`
	RoleName        string `json:"role_name"`
	DepartmentName  string `json:"department_name"`
	LoginURL        string `json:"login_url"`
	TempPassword    string `json:"temp_password"`
	ResetURL        string `json:"reset_url"`
	CooldownMinutes string `json:"cooldown_minutes"`
}
