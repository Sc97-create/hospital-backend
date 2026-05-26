package rolepermissions

import (
	"hospital-backend/internal/modules"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/roles"
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
	for _, each := range roles.DefaultRoleArr {
		switch each {
		case roles.DefaultRoleDoctor:
			if roleId, ok := roleMap[each]; ok {
				if moduleId, ok := moduleMap[modules.Appointment]; ok {
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, viewPermissionId, moduleId, organisationID))
					}
				}
				if moduleId, ok := moduleMap[modules.Prescription]; ok {
					if addPermissionId, ok := permissionMap[permissions.Create]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, addPermissionId, moduleId, organisationID))
					}
					if deletePermissionId, ok := permissionMap[permissions.Delete]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, deletePermissionId, moduleId, organisationID))
					}
					if editPermissionId, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, editPermissionId, moduleId, organisationID))
					}
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, viewPermissionId, moduleId, organisationID))
					}
				}
				if moduleId, ok := moduleMap[modules.Patient]; ok {
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, viewPermissionId, moduleId, organisationID))
					}
				}
				if moduleId, ok := moduleMap[modules.Patient]; ok {
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, viewPermissionId, moduleId, organisationID))
					}
					if updatePermissionId, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, updatePermissionId, moduleId, organisationID))
					}
				}
				if moduleId, ok := moduleMap[modules.Medicine]; ok {
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleId, viewPermissionId, moduleId, organisationID))
					}
				}
			}
		case roles.DefaultRoleAdmin:
			if roleID, ok := roleMap[each]; ok {
				if moduleId, ok := moduleMap[modules.Employee]; ok {
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, viewPermissionId, moduleId, organisationID))
					}
					if updatePermissionId, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, updatePermissionId, moduleId, organisationID))
					}
					if deletePermissionId, ok := permissionMap[permissions.Delete]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, deletePermissionId, moduleId, organisationID))
					}
					if addEmployeePermissionId, ok := permissionMap[permissions.Create]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, addEmployeePermissionId, moduleId, organisationID))
					}
					if addRole, ok := permissionMap[permissions.Create]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, addRole, moduleId, organisationID))
					}
					if viewRole, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, viewRole, moduleId, organisationID))
					}
					if deleteRole, ok := permissionMap[permissions.Delete]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, deleteRole, moduleId, organisationID))
					}
					if updateRole, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, updateRole, moduleId, organisationID))
					}
					if addDepartment, ok := permissionMap[permissions.Create]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, addDepartment, moduleId, organisationID))
					}
					if viewDepartment, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, viewDepartment, moduleId, organisationID))
					}
					if deleteDepartment, ok := permissionMap[permissions.Delete]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, deleteDepartment, moduleId, organisationID))
					}
					if updateDepartment, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, updateDepartment, moduleId, organisationID))

					}
					if updateLicense, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, updateLicense, moduleId, organisationID))
					}
				}
			}
		case roles.DefaultRolePharmacist:
			if roleID, ok := roleMap[each]; ok {
				if moduleId, ok := moduleMap[modules.Medicine]; ok {
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, viewPermissionId, moduleId, organisationID))
					}
					if updatePermissionId, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, updatePermissionId, moduleId, organisationID))
					}
					if deletePermissionId, ok := permissionMap[permissions.Delete]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, deletePermissionId, moduleId, organisationID))
					}
					if addPermissionId, ok := permissionMap[permissions.Create]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, addPermissionId, moduleId, organisationID))
					}
				}
				if moduleId, ok := moduleMap[modules.Prescription]; ok {
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, viewPermissionId, moduleId, organisationID))
					}
					if updatePermissionId, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, updatePermissionId, moduleId, organisationID))
					}
				}
				if moduleId, ok := moduleMap[modules.Billing]; ok {
					if viewPermissionId, ok := permissionMap[permissions.View]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, viewPermissionId, moduleId, organisationID))
					}
					if updatePermissionId, ok := permissionMap[permissions.Update]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, updatePermissionId, moduleId, organisationID))
					}
					if deletePermissionId, ok := permissionMap[permissions.Delete]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, deletePermissionId, moduleId, organisationID))
					}
					if addPermissionId, ok := permissionMap[permissions.Create]; ok {
						rolePermissions = append(rolePermissions, s.toRolePermModel(roleID, addPermissionId, moduleId, organisationID))
					}
				}
			}
		}

	}
	return rolePermissions
}

func (s *RolePermissionService) toRolePermModel(roleID string, permID string, moduleID string, organisationID string) RolePermission {
	return RolePermission{
		ID:             uuid.New().String(),
		RoleID:         roleID,
		PermissionID:   permID,
		ModuleID:       moduleID,
		CreatedAt:      time.Now(),
		OrganisationID: organisationID,
	}
}

/*
 */
