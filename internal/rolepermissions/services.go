package rolepermissions

import (
	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/rolepermissions/dto"
	"hospital-backend/internal/roles"
	wrapError "hospital-backend/shared/error"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RolePermissionService struct {
	DB   *gorm.DB
	repo RolePermissionRepo
}

func NewRolePermissionService(db *gorm.DB, repo RolePermissionRepo) *RolePermissionService {
	return &RolePermissionService{DB: db, repo: repo}
}

func (s *RolePermissionService) Create(rolePermission *RolePermission) error {
	return s.repo.Create(rolePermission)
}

func (s *RolePermissionService) FindModulesByRoleID(roleID string) (dto.RoleAccess, error) {
	if roleID == "" {
		return dto.RoleAccess{Permissions: []dto.RoleModulePermission{}}, nil
	}
	isAdmin, err := s.repo.IsAdminRole(roleID)
	if err != nil {
		return dto.RoleAccess{}, wrapError.ErrRolePermissionsFetchFailed
	}
	if isAdmin {
		return dto.RoleAccess{
			IsAdmin:     true,
			Permissions: []dto.RoleModulePermission{},
		}, nil
	}
	rows, err := s.repo.FindModulePermissionsByRoleID(roleID)
	if err != nil {
		return dto.RoleAccess{}, wrapError.ErrRolePermissionsFetchFailed
	}
	return dto.RoleAccess{
		IsAdmin:     false,
		Permissions: mapModulePermissionRows(rows),
	}, nil
}

func mapModulePermissionRows(rows []dto.ModulePermissionRow) []dto.RoleModulePermission {
	result := make([]dto.RoleModulePermission, 0, len(rows))
	for _, row := range rows {
		flags := dto.ModulePermissionFlags{}
		for _, name := range row.PermissionNames {
			applyPermissionFlag(&flags, name)
		}
		result = append(result, dto.RoleModulePermission{
			ModuleName:  row.ModuleName,
			Permissions: flags,
		})
	}
	return result
}

func applyPermissionFlag(flags *dto.ModulePermissionFlags, name string) {
	switch name {
	case permissions.Create:
		flags.Create = true
	case permissions.Update:
		flags.Update = true
	case permissions.View:
		flags.View = true
	case permissions.Delete:
		flags.Delete = true
	}
}

func (s *RolePermissionService) InsertMany(tx *gorm.DB, roleArr []roles.Role, permissionsArr []permissions.Permission, modulesArr []modules.Modules, organisationID string) error {
	rolePermissions := s.createRPModel(roleArr, permissionsArr, modulesArr, organisationID)
	err := s.repo.BatchCreate(tx, rolePermissions)
	if err != nil {
		return err
	}
	return nil
}

func (s *RolePermissionService) createRPModel(rolesArr []roles.Role, permissionsArr []permissions.Permission, modulesArr []modules.Modules, organisationID string) []RolePermission {
	roleMap := make(map[string]string)
	for _, each := range rolesArr {
		roleMap[each.Name] = each.ID
	}
	moduleMap := make(map[string]string)
	for _, each := range modulesArr {
		moduleMap[each.Name] = each.ID
	}
	permissionMap := make(map[string]string)
	for _, each := range permissionsArr {
		permissionMap[each.Name] = each.ID
	}

	var rolePermissions []RolePermission
	for _, roleName := range roles.DefaultRoleArr {
		roleID, ok := roleMap[roleName]
		if !ok {
			continue
		}
		if roleName == roles.DefaultRoleAdmin {
			rolePermissions = append(rolePermissions, s.toAdminRolePermModel(roleID, organisationID))
			continue
		}
		for _, grant := range defaultRolePermissionMatrix[roleName] {
			moduleID, ok := moduleMap[grant.Module]
			if !ok {
				continue
			}
			for _, action := range grant.Actions {
				permID, ok := permissionMap[action]
				if !ok {
					continue
				}
				rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, permID, moduleID, organisationID, false))
			}
		}
	}
	return rolePermissions
}

func (s *RolePermissionService) toRolePermModel(roleID string, permID string, moduleID string, organisationID string, isAdmin bool) RolePermission {
	if !isAdmin && permID == "" && moduleID == "" {
		return RolePermission{}
	}

	return RolePermission{
		ID:             uuid.New().String(),
		RoleID:         roleID,
		PermissionID:   nullableUUID(permID),
		ModuleID:       nullableUUID(moduleID),
		CreatedAt:      time.Now(),
		OrganisationID: organisationID,
		IsAdmin:        isAdmin,
	}
}

func (s *RolePermissionService) toAdminRolePermModel(roleID string, organisationID string) RolePermission {
	return RolePermission{
		ID:             uuid.New().String(),
		RoleID:         roleID,
		PermissionID:   nil,
		ModuleID:       nil,
		CreatedAt:      time.Now(),
		OrganisationID: organisationID,
		IsAdmin:        true,
	}
}

func nullableUUID(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
