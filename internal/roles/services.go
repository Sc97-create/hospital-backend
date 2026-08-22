package roles

import (
	"hospital-backend/internal/roles/dto"
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

func (RoleSer *RoleServices) FindMany(organisationID string, limit int, offset int) ([]dto.RoleResponse, int64, error) {
	roles, err := RoleSer.RoleRepo.FindMany(organisationID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := RoleSer.RoleRepo.Count(organisationID)
	if err != nil {
		return nil, 0, err
	}
	return RoleSer.arrayMapToRoleResponse(roles), total, nil
}

func (RoleSer *RoleServices) arrayMapToRoleResponse(roles []Role) []dto.RoleResponse {
	roleResponse := []dto.RoleResponse{}
	for _, each := range roles {
		roleResponse = append(roleResponse, RoleSer.mapToRoleResponse(each))
	}
	return roleResponse
}

func (RoleSer *RoleServices) mapToRoleResponse(role Role) dto.RoleResponse {
	return dto.RoleResponse{
		ID:   role.ID,
		Name: role.Name,
	}
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

func (RoleSer *RoleServices) FindByID(id string) (Role, error) {
	return RoleSer.RoleRepo.FindByID(id)
}
