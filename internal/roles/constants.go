package roles

const (
	DefaultRoleAdmin         = "Super Admin"
	DefaultRoleHospitalAdmin = "Hospital Admin"
	DefaultRoleDoctor        = "Doctor"
	DefaultRoleNurse         = "Nurse"
	DefaultRoleReceptionist  = "Receptionist"
	DefaultRoleLabTechnician = "Lab Technician"
	DefaultRolePharmacist    = "Pharmacist"
)

type RoleArr []string

var DefaultRoleArr RoleArr = RoleArr{
	DefaultRoleAdmin,
	DefaultRoleHospitalAdmin,
	DefaultRoleDoctor,
	DefaultRoleNurse,
	DefaultRoleReceptionist,
	DefaultRoleLabTechnician,
	DefaultRolePharmacist,
}
