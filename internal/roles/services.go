package roles

type RoleServices struct {
	RoleRepo RoleRepository
}

func NewRoleServices(RoleRepo RoleRepository) *RoleServices {
	return &RoleServices{RoleRepo: RoleRepo}
}

func (RoleSer *RoleServices) FindMany(limit int, offset int) ([]Role, error) {
	return RoleSer.RoleRepo.FindMany(limit, offset)
}
