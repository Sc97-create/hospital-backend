package dto

type EmployeeResponse struct {
	EmployeeID             string `json:"employee_id"`
	EmployeeName           string `json:"employee_name"`
	EmployeeFirstName      string `json:"employee_first_name"`
	EmployeeLastName       string `json:"employee_last_name"`
	EmployeeEmail          string `json:"employee_email"`
	EmployeePhone          string `json:"employee_phone"`
	EmployeeRoleID         string `json:"employee_role_id"`
	EmployeeDepartmentID   string `json:"employee_department_id"`
	EmployeeStatus         string `json:"employee_status"`
	EmployeeOrganisationID string `json:"employee_organisation_id"`
}

type EmployeeListResponse struct {
	Data  []EmployeeResponse `json:"data"`
	Total int64              `json:"total"`
	Code  int                `json:"code"`
}
