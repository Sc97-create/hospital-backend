package rolepermissions

import "gorm.io/gorm"

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
