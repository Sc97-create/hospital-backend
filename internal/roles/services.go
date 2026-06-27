package roles

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleServices struct {
	RoleRepo RoleRepository
}

func NewRoleServices(RoleRepo RoleRepository) *RoleServices {
	return &RoleServices{RoleRepo: RoleRepo}
}

func (RoleSer *RoleServices) FindMany(limit int, offset int) ([]Role, error) {
	return RoleSer.RoleRepo.FindMany(limit, offset)
}
func (RoleSer *RoleServices) FindRoleByOrgID(organisationID string) ([]Role, error) {
	roles, err := RoleSer.FindRoleByOrgID(organisationID)
	if err != nil {
		return nil, err
	}
	return roles, nil
}
func (RoleSer *RoleServices) InsertMany(tx *gorm.DB, organisationID string) ([]Role, error) {
	role := RoleSer.createRoleArray(organisationID)
	err := RoleSer.RoleRepo.InsertMany(tx, role)
	if err != nil {
		return nil, err
	}
	return role, nil
}
func (RoleSer *RoleServices) createRoleArray(organisationID string) []Role {
	var defaultroles []Role
	for _, each := range DefaultRoleArr {
		defaultroles = append(defaultroles, Role{
			ID:             uuid.NewString(),
			Name:           each,
			CreatedAt:      time.Now(),
			OrganisationID: organisationID,
		})
	}
	return defaultroles
}
func (RoleSer *RoleServices) FindRoleByNames(organisationID string, name string) (Role, error) {
	return RoleSer.RoleRepo.FindRoleByNames(organisationID, name)
}
