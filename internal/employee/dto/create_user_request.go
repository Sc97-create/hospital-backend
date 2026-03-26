package dto

type EmpRequest struct {
	OrganisationID    string `json:"organisation_id"`
	UserName          string `json:"username"`
	EmailID           string `json:"email_id"`
	RoleID            string `json:"role_id"`
	DepartmentID      string `json:"department_id"`
	PhoneNumber       string `json:"phone_number"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Password          string `json:"password"`
	ConfirmPassword   string `json:"confirm_password"`
	IsConsent         bool   `json:"is_consent"`
	AssignDefRoles    bool   `json:"assign_default_roles"`
	AssignDefDept     bool   `json:"assign_default_departments"`
	AssignDefRolePerm bool   `json:"assign_default_role_permissions"`
}
