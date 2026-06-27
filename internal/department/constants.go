package department

const (
	DefaultDeptAdmin           = "Administration"
	DefaultDeptEmergency       = "Emergency"
	DefaultDeptGeneralMedicine = "General Medicine"
	DefaultDeptCardiology      = "Cardiology"
	DefaultDeptRadiology       = "Radiology"
	DefaultDeptLaboratory      = "Laboratory"
	DefaultDeptPharmacy        = "Pharmacy"
	DefaultDeptNursing         = "Nursing"
)

type DeptArr []string

var DefaultDeptArr DeptArr = DeptArr{
	DefaultDeptAdmin,
	DefaultDeptEmergency,
	DefaultDeptGeneralMedicine,
	DefaultDeptCardiology,
	DefaultDeptRadiology,
	DefaultDeptLaboratory,
	DefaultDeptPharmacy,
	DefaultDeptNursing,
}
