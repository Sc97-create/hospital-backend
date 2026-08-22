package dto

type EmployeeResponse struct {
	EmployeeID             string `json:"employee_id"`
	EmployeeCode           string `json:"employee_code"`
	EmployeeName           string `json:"employee_name"`
	EmployeeFirstName      string `json:"employee_first_name"`
	EmployeeLastName       string `json:"employee_last_name"`
	EmployeeEmail          string `json:"employee_email"`
	EmployeePhone          string `json:"employee_phone"`
	EmployeeRoleID         string `json:"employee_role_id"`
	RoleName               string `json:"role_name"`
	EmployeeDepartmentID   string `json:"employee_department_id"`
	DepartmentName         string `json:"department_name"`
	EmployeeStatus         string `json:"employee_status"`
	EmployeeOrganisationID string `json:"employee_organisation_id"`
}

type EmployeeListResponse struct {
	Data       []EmployeeResponse `json:"data"`
	Total      int64              `json:"total"`
	TotalCount int64              `json:"total_count"`
	Code       int                `json:"code"`
}
