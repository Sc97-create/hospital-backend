package organisation

import (
	"hospital-backend/internal/department"
	"hospital-backend/internal/license"
	dto "hospital-backend/internal/organisation/DTO"
	"hospital-backend/internal/permissions"
	"hospital-backend/internal/rolepermissions"
	"hospital-backend/internal/roles"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// service should not depend on internal repo field
type OrganisationService struct {
	DB                    *gorm.DB
	OrganisationRepo      OrganisationRepo
	LicenseRep            *license.LicenseService
	RoleServices          *roles.RoleServices
	DeptServices          *department.DepartmentService
	PermService           *permissions.PermService
	RolePermissionService *rolepermissions.RolePermissionService
}

func NewOrganisationService(db *gorm.DB, orgRepo OrganisationRepo, license *license.LicenseService, roleRepo *roles.RoleServices, deptRepo *department.DepartmentService, permServ *permissions.PermService, rolePermissionRepo *rolepermissions.RolePermissionService) *OrganisationService {
	return &OrganisationService{DB: db, OrganisationRepo: orgRepo, LicenseRep: license, RoleServices: roleRepo, DeptServices: deptRepo, PermService: permServ, RolePermissionService: rolePermissionRepo}
}

func (OService *OrganisationService) CreateOrganisation(payloadRequest dto.OrganisationPayload) (string, error) {
	organisation := OService.createOrgModel(payloadRequest)
	modules, permissions, err := OService.PermService.FindMany()
	if err != nil {
		return "", err
	}
	err = OService.DB.Transaction(func(tx *gorm.DB) error {
		if err := OService.OrganisationRepo.Create(tx, organisation); err != nil {
			return err
		}
		if err := OService.LicenseRep.CreateLicenseSrv(tx, organisation.OrganisationName, 6, organisation.ID, "month", time.Now()); err != nil {
			return err
		}
		roles, err := OService.RoleServices.InsertMany(tx, organisation.ID)
		if err != nil {
			return err
		}
		if err := OService.DeptServices.InsertMany(tx, organisation.ID); err != nil {
			return err
		}
		if err := OService.RolePermissionService.InsertMany(tx, roles, permissions, modules, organisation.ID); err != nil {
			return err
		}

		//getpermissions
		return nil
	})
	if err != nil {
		return "", err
	}

	return organisation.ID, nil
}
func (Oservice *OrganisationService) createOrgModel(payloadReq dto.OrganisationPayload) Organisation {
	var organisation Organisation
	organisation.ID = uuid.NewString()
	organisation.OrganisationName = payloadReq.OrganisationName
	organisation.LegalEntityName = payloadReq.LegalEntityName
	organisation.HospitalType = payloadReq.HospitalType
	organisation.CreatedAt = time.Now()
	organisation.UpdatedAt = time.Now()
	organisation.Code = uuid.NewString()
	return organisation
}
func (OService *OrganisationService) UpdateOrganisationLoc(payloadReques dto.OrganisationPayload) (err error) {
	organisation := new(Organisation)
	organisation.ID = payloadReques.OrganisationID
	organisation.Address.CityID = payloadReques.City
	organisation.Address.StateID = payloadReques.State
	organisation.Address.CountryID = payloadReques.Country
	organisation.Security.EnableAuditLog = payloadReques.AuditLogs
	organisation.Security.EmergencyAccess = payloadReques.EmergencyAcess
	err = OService.OrganisationRepo.UpdateLocationByID(organisation)
	if err != nil {
		return

	}
	return
}
func (Oservice *OrganisationService) GetByID(organisationID string) (Organisation, error) {
	Organisation, err := Oservice.OrganisationRepo.GetOrganisationByID(organisationID)
	if err != nil {
		return Organisation, err
	}
	return Organisation, nil
}
func (Oservice *OrganisationService) Update(organisationID string, payload dto.OrganisationPayload) (err error) {
	updateMap := make(map[string]interface{})
	if payload.OrganisationName != "" {
		updateMap["organisation_name"] = payload.OrganisationName
	}
	if payload.LegalEntityName != "" {
		updateMap["legal_entity_name"] = payload.LegalEntityName
	}
	if payload.HospitalType != "" {
		updateMap["hospital_type"] = payload.HospitalType
	}
	updateMap["updated_at"] = time.Now()
	err = Oservice.OrganisationRepo.Update(organisationID, updateMap)
	if err != nil {
		return
	}
	return
}
