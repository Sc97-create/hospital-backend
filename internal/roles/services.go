package roles

import (
	"hospital-backend/internal/roles/dto"
	wrapError "hospital-backend/shared/error"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RoleServices struct {
	RoleRepo RoleRepository
}

func NewRoleServices(RoleRepo RoleRepository) *RoleServices {
	return &RoleServices{RoleRepo: RoleRepo}
}

func (RoleSer *RoleServices) FindMany(log *zap.Logger, organisationID string, limit int, offset int) ([]dto.RoleResponse, int64, error) {
	log = ensureLog(log)
	roles, err := RoleSer.RoleRepo.FindMany(log, organisationID, limit, offset)
	if err != nil {
		log.Error("role list failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_list"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrRolesFetchFailed
	}
	total, err := RoleSer.RoleRepo.Count(log, organisationID)
	if err != nil {
		log.Error("role list failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrRolesFetchFailed
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

func (RoleSer *RoleServices) FindRoleByOrgID(log *zap.Logger, organisationID string) ([]Role, error) {
	log = ensureLog(log)
	roles, err := RoleSer.RoleRepo.FindRoleByOrgID(log, organisationID)
	if err != nil {
		log.Error("role lookup failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, err
	}
	return roles, nil
}

func (RoleSer *RoleServices) InsertMany(log *zap.Logger, tx *gorm.DB, organisationID string) ([]Role, error) {
	log = ensureLog(log)
	role := RoleSer.createRoleArray(organisationID)
	err := RoleSer.RoleRepo.InsertMany(log, tx, role)
	if err != nil {
		log.Error("role seed failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_insert"),
			zap.Error(err),
		)
		return nil, err
	}
	log.Info("role seed success",
		zap.String("organisation_id", organisationID),
		zap.Int("count", len(role)),
	)
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

func (RoleSer *RoleServices) FindRoleByNames(log *zap.Logger, organisationID string, name string) (Role, error) {
	log = ensureLog(log)
	role, err := RoleSer.RoleRepo.FindRoleByNames(log, organisationID, name)
	if err != nil {
		log.Error("role lookup failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return Role{}, err
	}
	return role, nil
}

func (RoleSer *RoleServices) FindByID(log *zap.Logger, id string) (Role, error) {
	log = ensureLog(log)
	role, err := RoleSer.RoleRepo.FindByID(log, id)
	if err != nil {
		log.Error("role lookup failed",
			zap.String("role_id", id),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return Role{}, err
	}
	return role, nil
}
