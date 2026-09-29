package tenants

import (
	"errors"
	"strings"
	"time"

	"hospital-backend/central/customers"
	"hospital-backend/central/organisations"
	orgdto "hospital-backend/central/organisations/dto"
	dto "hospital-backend/central/tenants/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OrganisationProvisioner is the organisation-service surface used by tenants.
type OrganisationProvisioner interface {
	HasOrganisationsForTenant(log *zap.Logger, tx *gorm.DB, tenantID string) (bool, error)
	AddOrganisationTx(log *zap.Logger, tx *gorm.DB, payload orgdto.OrganisationPayload) (string, error)
	GetPrimaryByTenantID(log *zap.Logger, tenantID string) (organisations.Organisation, error)
	GetOrgByID(log *zap.Logger, organisationID string) (organisations.Organisation, error)
	Update(log *zap.Logger, organisationID string, payload orgdto.OrganisationPayload) error
}

// CustomerTenantBinder assigns tenant_id onto an existing customer.
type CustomerTenantBinder interface {
	BindTenantID(log *zap.Logger, tx *gorm.DB, customerID, tenantID string) error
}

type TenantService struct {
	DB             *gorm.DB
	Repo           TenantRepository
	OrgSvc         OrganisationProvisioner
	CustomerBinder CustomerTenantBinder
	Admin          organisations.HospitalAdminProvisioner
}

func NewTenantService(db *gorm.DB, repo TenantRepository, orgSvc OrganisationProvisioner, customerBinder CustomerTenantBinder) *TenantService {
	return &TenantService{DB: db, Repo: repo, OrgSvc: orgSvc, CustomerBinder: customerBinder}
}

func (s *TenantService) CreateTenant(log *zap.Logger, customerID string, payload dto.CreateTenantPayload) (dto.CreateTenantResult, error) {
	log = ensureLog(log)
	if strings.TrimSpace(customerID) == "" {
		log.Warn("tenant create failed", zap.String("reason", "missing_customer_id"))
		return dto.CreateTenantResult{}, wrapError.ErrInvalidRequest
	}
	if err := validateCreatePayload(payload); err != nil {
		log.Warn("tenant create failed", zap.String("reason", "validation"), zap.Error(err))
		return dto.CreateTenantResult{}, wrapError.ErrInvalidRequest
	}

	tx := s.DB.Begin()
	if tx.Error != nil {
		log.Error("tenant create failed", zap.String("reason", "begin_tx"), zap.Error(tx.Error))
		return dto.CreateTenantResult{}, wrapError.ErrTenantCreateFailed
	}

	result, err := s.createTenantInTx(log, tx, customerID, payload)
	if err != nil {
		_ = tx.Rollback()
		return dto.CreateTenantResult{}, err
	}
	if err := tx.Commit().Error; err != nil {
		log.Error("tenant create failed", zap.String("reason", "commit_tx"), zap.Error(err))
		return dto.CreateTenantResult{}, wrapError.ErrTenantCreateFailed
	}
	if err := s.provisionHospitalAdmin(log, result); err != nil {
		return dto.CreateTenantResult{}, err
	}
	return result, nil
}

func (s *TenantService) provisionHospitalAdmin(log *zap.Logger, result dto.CreateTenantResult) error {
	if s.Admin == nil {
		return nil
	}
	if err := s.Admin.ProvisionHospitalAdmin(log, result.TenantID, result.OrganisationID); err != nil {
		log.Error("tenant create failed",
			zap.String("tenant_id", result.TenantID),
			zap.String("organisation_id", result.OrganisationID),
			zap.String("reason", "hospital_admin"),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (s *TenantService) createTenantInTx(log *zap.Logger, tx *gorm.DB, customerID string, payload dto.CreateTenantPayload) (dto.CreateTenantResult, error) {
	tenant := buildTenant(payload)
	if err := s.Repo.Create(log, tx, tenant); err != nil {
		log.Error("tenant create failed", zap.String("reason", "db_create"), zap.Error(err))
		return dto.CreateTenantResult{}, wrapError.ErrTenantCreateFailed
	}

	hasOrg, err := s.OrgSvc.HasOrganisationsForTenant(log, tx, tenant.ID)
	if err != nil {
		return dto.CreateTenantResult{}, err
	}
	if hasOrg {
		log.Warn("tenant create restricted",
			zap.String("tenant_id", tenant.ID),
			zap.String("reason", "organisation_exists"),
		)
		return dto.CreateTenantResult{}, wrapError.ErrTenantAlreadyHasOrganisation
	}

	orgID, err := s.OrgSvc.AddOrganisationTx(log, tx, buildPrimaryOrgPayload(tenant.ID, payload))
	if err != nil {
		return dto.CreateTenantResult{}, err
	}

	if err := s.bindTenantToCustomer(log, tx, customerID, tenant.ID); err != nil {
		return dto.CreateTenantResult{}, err
	}

	log.Info("tenant create success",
		zap.String("tenant_id", tenant.ID),
		zap.String("organisation_id", orgID),
		zap.String("customer_id", customerID),
	)
	return dto.CreateTenantResult{TenantID: tenant.ID, OrganisationID: orgID}, nil
}

func (s *TenantService) bindTenantToCustomer(log *zap.Logger, tx *gorm.DB, customerID, tenantID string) error {
	err := s.CustomerBinder.BindTenantID(log, tx, customerID, tenantID)
	if err == nil {
		return nil
	}
	if errors.Is(err, customers.ErrCustomerNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
		log.Warn("tenant create failed",
			zap.String("customer_id", customerID),
			zap.String("tenant_id", tenantID),
			zap.String("reason", "customer_not_found"),
		)
		return wrapError.ErrCustomerNotFound
	}
	log.Error("tenant create failed",
		zap.String("customer_id", customerID),
		zap.String("tenant_id", tenantID),
		zap.String("reason", "bind_customer_tenant"),
		zap.Error(err),
	)
	return wrapError.ErrCustomerUpdateFailed
}

// GetTenantByID returns the tenant and its primary organisation (is_primary=true).
func (s *TenantService) GetTenantByID(log *zap.Logger, tenantID string) (dto.TenantDetailResult, error) {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return dto.TenantDetailResult{}, wrapError.ErrInvalidRequest
	}

	tenant, err := s.Repo.GetByID(log, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("tenant get failed", zap.String("tenant_id", tenantID), zap.String("reason", "not_found"))
			return dto.TenantDetailResult{}, wrapError.ErrTenantNotFound
		}
		log.Error("tenant get failed", zap.String("tenant_id", tenantID), zap.String("reason", "db_read"), zap.Error(err))
		return dto.TenantDetailResult{}, wrapError.ErrTenantUpdateFailed
	}

	org, err := s.OrgSvc.GetPrimaryByTenantID(log, tenantID)
	if err != nil {
		return dto.TenantDetailResult{}, err
	}

	log.Info("tenant get success",
		zap.String("tenant_id", tenant.ID),
		zap.String("organisation_id", org.ID),
	)
	return dto.TenantDetailResult{
		Tenant:       toTenantView(tenant),
		Organisation: toOrganisationView(org),
	}, nil
}

// UpdateTenantOrg updates tenant and organisation using the given organisation_id.
func (s *TenantService) UpdateTenantOrg(log *zap.Logger, payload dto.UpdateTenantOrgPayload) (dto.TenantDetailResult, error) {
	log = ensureLog(log)
	if err := validateUpdatePayload(payload); err != nil {
		log.Warn("tenant update failed", zap.String("reason", "validation"), zap.Error(err))
		return dto.TenantDetailResult{}, wrapError.ErrInvalidRequest
	}

	org, err := s.OrgSvc.GetOrgByID(log, payload.OrganisationID)
	if err != nil {
		return dto.TenantDetailResult{}, err
	}

	if err := s.applyTenantUpdates(log, org.TenantID, payload); err != nil {
		return dto.TenantDetailResult{}, err
	}
	if hasOrganisationChanges(payload) {
		if err := s.OrgSvc.Update(log, payload.OrganisationID, toOrgUpdatePayload(payload)); err != nil {
			return dto.TenantDetailResult{}, err
		}
	}

	return s.reloadTenantDetail(log, org.TenantID, payload.OrganisationID)
}

func (s *TenantService) applyTenantUpdates(log *zap.Logger, tenantID string, payload dto.UpdateTenantOrgPayload) error {
	updates := map[string]interface{}{}
	if name := strings.TrimSpace(payload.LegalEntityName); name != "" {
		updates["name"] = name
	}
	if status := strings.TrimSpace(payload.TenantStatus); status != "" {
		updates["status"] = strings.ToLower(status)
	}
	if len(updates) == 0 {
		return nil
	}
	if err := s.Repo.UpdateByID(log, nil, tenantID, updates); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("tenant update failed", zap.String("tenant_id", tenantID), zap.String("reason", "not_found"))
			return wrapError.ErrTenantNotFound
		}
		log.Error("tenant update failed", zap.String("tenant_id", tenantID), zap.String("reason", "db_update"), zap.Error(err))
		return wrapError.ErrTenantUpdateFailed
	}
	return nil
}

func (s *TenantService) reloadTenantDetail(log *zap.Logger, tenantID, organisationID string) (dto.TenantDetailResult, error) {
	tenant, err := s.Repo.GetByID(log, tenantID)
	if err != nil {
		log.Error("tenant update reload failed",
			zap.String("tenant_id", tenantID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.TenantDetailResult{}, wrapError.ErrTenantUpdateFailed
	}
	updatedOrg, err := s.OrgSvc.GetOrgByID(log, organisationID)
	if err != nil {
		return dto.TenantDetailResult{}, err
	}
	log.Info("tenant update success",
		zap.String("tenant_id", tenant.ID),
		zap.String("organisation_id", updatedOrg.ID),
	)
	return dto.TenantDetailResult{
		Tenant:       toTenantView(tenant),
		Organisation: toOrganisationView(updatedOrg),
	}, nil
}

// Activate marks a tenant active after payment/webhook (or free-trial) confirmation.
// Idempotent when the tenant is already active.
func (s *TenantService) Activate(log *zap.Logger, tenantID string) error {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return wrapError.ErrInvalidRequest
	}
	if err := s.Repo.UpdateStatusByID(log, nil, tenantID, constants.StatusActive); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("tenant activate failed",
				zap.String("tenant_id", tenantID),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrTenantNotFound
		}
		log.Error("tenant activate failed",
			zap.String("tenant_id", tenantID),
			zap.String("reason", "update"),
			zap.Error(err),
		)
		return wrapError.ErrTenantUpdateFailed
	}
	log.Info("tenant activate success", zap.String("tenant_id", tenantID))
	return nil
}

func buildTenant(payload dto.CreateTenantPayload) Tenant {
	now := time.Now()
	return Tenant{
		ID:        uuid.NewString(),
		Name:      strings.TrimSpace(payload.LegalEntityName),
		Status:    constants.StatusPending, // activated after paid subscription / free-trial provision
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func buildPrimaryOrgPayload(tenantID string, payload dto.CreateTenantPayload) orgdto.OrganisationPayload {
	return orgdto.OrganisationPayload{
		TenantID:         tenantID,
		IsPrimary:        true,
		LegalEntityName:  strings.TrimSpace(payload.LegalEntityName),
		OrganisationType: strings.ToLower(strings.TrimSpace(payload.HospitalType)),
		FacilityName:     strings.TrimSpace(payload.FacilityName),
		Address1:         strings.TrimSpace(payload.FacilityAddress.Address1),
		Address2:         strings.TrimSpace(payload.FacilityAddress.Address2),
		City:             strings.TrimSpace(payload.FacilityAddress.City),
		State:            strings.TrimSpace(payload.FacilityAddress.State),
		Status:           constants.StatusActive,
	}
}

func toOrgUpdatePayload(payload dto.UpdateTenantOrgPayload) orgdto.OrganisationPayload {
	return orgdto.OrganisationPayload{
		OrganisationID:   payload.OrganisationID,
		LegalEntityName:  strings.TrimSpace(payload.LegalEntityName),
		OrganisationType: strings.TrimSpace(payload.HospitalType),
		FacilityName:     strings.TrimSpace(payload.FacilityName),
		RegistrationNo:   strings.TrimSpace(payload.RegistrationNo),
		LicenseNumber:    strings.TrimSpace(payload.LicenseNumber),
		GSTIN:            strings.TrimSpace(payload.GSTIN),
		Address1:         strings.TrimSpace(payload.FacilityAddress.Address1),
		Address2:         strings.TrimSpace(payload.FacilityAddress.Address2),
		City:             strings.TrimSpace(payload.FacilityAddress.City),
		State:            strings.TrimSpace(payload.FacilityAddress.State),
		Status:           strings.TrimSpace(payload.Status),
	}
}

func toTenantView(tenant Tenant) dto.TenantView {
	return dto.TenantView{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Status:    tenant.Status,
		CreatedAt: tenant.CreatedAt.Format(time.RFC3339),
		UpdatedAt: tenant.UpdatedAt.Format(time.RFC3339),
	}
}

func toOrganisationView(org organisations.Organisation) dto.OrganisationView {
	return dto.OrganisationView{
		ID:               org.ID,
		TenantID:         org.TenantID,
		IsPrimary:        org.IsPrimary,
		LegalEntityName:  org.LegalEntityName,
		OrganisationType: org.OrganisationType,
		FacilityName:     org.FacilityName,
		RegistrationNo:   org.RegistrationNo,
		LicenseNumber:    org.LicenseNumber,
		GSTIN:            org.GSTIN,
		Address: dto.FacilityAddress{
			Address1: org.Address.Address1,
			Address2: org.Address.Address2,
			City:     org.Address.City,
			State:    org.Address.State,
		},
		Status: org.Status,
	}
}
