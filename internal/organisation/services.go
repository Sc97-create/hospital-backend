package organisation

import (
	"errors"
	dto "hospital-backend/internal/organisation/DTO"
	wrapError "hospital-backend/shared/error"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrganisationService struct {
	DB                    *gorm.DB
	OrganisationRepo      OrganisationRepo
	LicenseRep            LicenseCreator
	RoleServices          RoleSeeder
	DeptServices          DepartmentSeeder
	PermService           PermissionCatalogLookup
	RolePermissionService RolePermissionSeeder
}

func NewOrganisationService(db *gorm.DB, orgRepo OrganisationRepo, license LicenseCreator, roleRepo RoleSeeder, deptRepo DepartmentSeeder, permServ PermissionCatalogLookup, rolePermissionRepo RolePermissionSeeder) *OrganisationService {
	return &OrganisationService{DB: db, OrganisationRepo: orgRepo, LicenseRep: license, RoleServices: roleRepo, DeptServices: deptRepo, PermService: permServ, RolePermissionService: rolePermissionRepo}
}

func (OService *OrganisationService) CreateOrganisation(log *zap.Logger, payloadRequest dto.OrganisationPayload) (string, error) {
	log = ensureLog(log)
	organisation := OService.createOrgModel(payloadRequest)
	modules, permissions, err := OService.PermService.FindMany()
	if err != nil {
		log.Error("organisation create failed",
			zap.String("reason", "permissions_lookup"),
			zap.Error(err),
		)
		return "", wrapError.ErrOrganisationCreateFailed
	}

	err = OService.DB.Transaction(func(tx *gorm.DB) error {
		if err := OService.OrganisationRepo.Create(log, tx, organisation); err != nil {
			log.Error("organisation create failed",
				zap.String("reason", "db_create"),
				zap.Error(err),
			)
			return err
		}
		if err := OService.LicenseRep.CreateLicenseSrv(log, tx, organisation.OrganisationName, 6, organisation.ID, "month", time.Now()); err != nil {
			log.Error("organisation create failed",
				zap.String("organisation_id", organisation.ID),
				zap.String("reason", "license_create"),
				zap.Error(err),
			)
			return err
		}
		roles, err := OService.RoleServices.InsertMany(tx, organisation.ID)
		if err != nil {
			log.Error("organisation create failed",
				zap.String("organisation_id", organisation.ID),
				zap.String("reason", "roles_seed"),
				zap.Error(err),
			)
			return err
		}
		if err := OService.DeptServices.InsertMany(tx, organisation.ID); err != nil {
			log.Error("organisation create failed",
				zap.String("organisation_id", organisation.ID),
				zap.String("reason", "depts_seed"),
				zap.Error(err),
			)
			return err
		}
		if err := OService.RolePermissionService.InsertMany(tx, roles, permissions, modules, organisation.ID); err != nil {
			log.Error("organisation create failed",
				zap.String("organisation_id", organisation.ID),
				zap.String("reason", "role_permissions_seed"),
				zap.Error(err),
			)
			return err
		}
		return nil
	})
	if err != nil {
		return "", wrapError.ErrOrganisationCreateFailed
	}

	log.Info("organisation create success",
		zap.String("organisation_id", organisation.ID),
		zap.String("code", organisation.Code),
		zap.String("hospital_type", organisation.HospitalType),
		zap.Bool("license_created", true),
		zap.Bool("roles_seeded", true),
		zap.Bool("depts_seeded", true),
	)
	return organisation.ID, nil
}

func (OService *OrganisationService) createOrgModel(payloadReq dto.OrganisationPayload) Organisation {
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

func (OService *OrganisationService) UpdateOrganisationLoc(log *zap.Logger, payloadReques dto.OrganisationPayload) error {
	log = ensureLog(log)
	organisation := new(Organisation)
	organisation.ID = payloadReques.OrganisationID
	organisation.Address.CityID = payloadReques.City
	organisation.Address.StateID = payloadReques.State
	organisation.Address.CountryID = payloadReques.Country
	organisation.Security.EnableAuditLog = payloadReques.AuditLogs
	organisation.Security.EmergencyAccess = payloadReques.EmergencyAcess

	err := OService.OrganisationRepo.UpdateLocationByID(log, organisation)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("organisation location update failed",
				zap.String("organisation_id", payloadReques.OrganisationID),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrOrganisationNotFound
		}
		log.Error("organisation location update failed",
			zap.String("organisation_id", payloadReques.OrganisationID),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrOrganisationUpdateFailed
	}

	log.Info("organisation location update success",
		zap.String("organisation_id", payloadReques.OrganisationID),
		zap.Bool("enable_audit_logs", payloadReques.AuditLogs),
		zap.Bool("emergency_access", payloadReques.EmergencyAcess),
	)
	return nil
}

func (OService *OrganisationService) GetOrgByID(log *zap.Logger, organisationID string) (Organisation, error) {
	log = ensureLog(log)
	log.Debug("organisation get by id", zap.String("organisation_id", organisationID))

	org, err := OService.OrganisationRepo.GetOrganisationByID(log, organisationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("organisation get by id failed",
				zap.String("organisation_id", organisationID),
				zap.String("reason", "not_found"),
			)
			return Organisation{}, wrapError.ErrOrganisationNotFound
		}
		log.Error("organisation get by id failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return Organisation{}, wrapError.ErrOrganisationFetchFailed
	}

	log.Debug("organisation get by id success",
		zap.String("organisation_id", org.ID),
		zap.String("hospital_type", org.HospitalType),
	)
	return org, nil
}

func (OService *OrganisationService) Update(log *zap.Logger, organisationID string, payload dto.OrganisationPayload) error {
	log = ensureLog(log)
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

	err := OService.OrganisationRepo.Update(log, organisationID, updateMap)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("organisation update failed",
				zap.String("organisation_id", organisationID),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrOrganisationNotFound
		}
		log.Error("organisation update failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrOrganisationUpdateFailed
	}

	log.Info("organisation update success",
		zap.String("organisation_id", organisationID),
		zap.String("hospital_type", payload.HospitalType),
	)
	return nil
}
