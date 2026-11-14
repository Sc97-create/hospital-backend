package department

const DefaultDeptAdmin = "Administration"

type DeptArr []string

// DefaultDeptArr is the staff department created for an organisation admin.
// Clinical departments are master data and are seeded with their facility type.
var DefaultDeptArr DeptArr = DeptArr{
	DefaultDeptAdmin,
}
