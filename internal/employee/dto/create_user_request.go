package dto

type ShiftTimings struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type EmergencyDetails struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Contact string `json:"contact"`
}

type EmpRequest struct {
	OrganisationID   string           `json:"organisation_id"`
	UserName         string           `json:"username"`
	EmailID          string           `json:"email_id"`
	RoleID           string           `json:"role_id"`
	DepartmentID     string           `json:"dept_id"`
	PhoneNumber      string           `json:"phone_number"`
	MobileNumber     string           `json:"mobile_number"`
	FirstName        string           `json:"first_name"`
	LastName         string           `json:"last_name"`
	Password         string           `json:"password"`
	ConfirmPassword  string           `json:"confirm_password"`
	IsConsent        bool             `json:"is_consent"`
	Address          string           `json:"address"`
	DateOfBirth      string           `json:"date_of_birth"`
	DateOfJoining    string           `json:"date_of_joining"`
	LicenseNo        string           `json:"license_no"`
	Qualification    string           `json:"qualification"`
	EmployeeType     string           `json:"employee_type"`
	ShiftTimings     ShiftTimings     `json:"shift_timings"`
	EmergencyDetails EmergencyDetails `json:"emergency_details"`
}
