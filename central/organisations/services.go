package organisations

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	dto "hospital-backend/central/organisations/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HospitalAdminProvisioner creates the tenant customer as Hospital Admin after an organisation exists.
type HospitalAdminProvisioner interface {
	ProvisionHospitalAdmin(log *zap.Logger, tenantID, organisationID string) error
}

type OrganisationService struct {
	DB               *gorm.DB
	OrganisationRepo OrganisationRepo
	Admin            HospitalAdminProvisioner
}

func NewOrganisationService(db *gorm.DB, orgRepo OrganisationRepo) *OrganisationService {
	return &OrganisationService{DB: db, OrganisationRepo: orgRepo}
}

func (s *OrganisationService) AddOrganisation(log *zap.Logger, payload dto.OrganisationPayload) (string, error) {
	return s.addOrganisation(log, nil, payload)
}

// AddOrganisationTx creates an organisation inside an existing transaction.
func (s *OrganisationService) AddOrganisationTx(log *zap.Logger, tx *gorm.DB, payload dto.OrganisationPayload) (string, error) {
	if tx == nil {
		return "", wrapError.ErrInvalidRequest
	}
	return s.addOrganisation(log, tx, payload)
}

// HasOrganisationsForTenant reports whether any organisation exists for tenantID.
// When tx is non-nil, the check runs inside that transaction.
func (s *OrganisationService) HasOrganisationsForTenant(log *zap.Logger, tx *gorm.DB, tenantID string) (bool, error) {
	log = ensureLog(log)
	if strings.TrimSpace(tenantID) == "" {
		return false, wrapError.ErrInvalidRequest
	}
	db := s.DB
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.Model(&Organisation{}).Where("tenant_id = ?", tenantID).Count(&count).Error
	if err != nil {
		log.Error("organisation tenant membership check failed",
			zap.String("tenant_id", tenantID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return false, wrapError.ErrOrganisationFetchFailed
	}
	return count > 0, nil
}

func (s *OrganisationService) addOrganisation(log *zap.Logger, tx *gorm.DB, payload dto.OrganisationPayload) (string, error) {
	log = ensureLog(log)
	if err := validateAddPayload(payload); err != nil {
		log.Warn("organisation add failed", zap.String("reason", "validation"), zap.Error(err))
		return "", wrapError.ErrInvalidRequest
	}

	org := s.buildOrgModel(payload)
	if err := s.OrganisationRepo.Create(log, tx, org); err != nil {
		log.Error("organisation add failed",
			zap.String("tenant_id", org.TenantID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", wrapError.ErrOrganisationCreateFailed
	}

	if err := s.provisionHospitalAdmin(log, tx, &org); err != nil {
		return "", err
	}

	log.Info("organisation add success",
		zap.String("organisation_id", org.ID),
		zap.String("tenant_id", org.TenantID),
		zap.String("organisation_type", org.OrganisationType),
		zap.Bool("is_primary", org.IsPrimary),
	)
	return org.ID, nil
}

func (s *OrganisationService) provisionHospitalAdmin(log *zap.Logger, tx *gorm.DB, org *Organisation) error {
	if tx != nil || s.Admin == nil {
		return nil
	}
	if err := s.Admin.ProvisionHospitalAdmin(log, org.TenantID, org.ID); err != nil {
		log.Error("organisation add failed",
			zap.String("organisation_id", org.ID),
			zap.String("tenant_id", org.TenantID),
			zap.String("reason", "hospital_admin"),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (s *OrganisationService) GetOrgByID(log *zap.Logger, organisationID string) (Organisation, error) {
	log = ensureLog(log)
	if strings.TrimSpace(organisationID) == "" {
		return Organisation{}, wrapError.ErrInvalidRequest
	}

	org, err := s.OrganisationRepo.GetByID(log,
		`select id,tenant_id,is_primary,legal_entity_name,organisation_type,facility_name,registration_no,license_number,license_expiry,gstin,address,data_sharing,status,created_at,updated_at from organisations where id=$1`,
		organisationID,
	)
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
	return org, nil
}

// GetPrimaryByTenantID returns the primary organisation for a tenant.
func (s *OrganisationService) GetPrimaryByTenantID(log *zap.Logger, tenantID string) (Organisation, error) {
	log = ensureLog(log)
	if strings.TrimSpace(tenantID) == "" {
		return Organisation{}, wrapError.ErrInvalidRequest
	}

	org, err := s.OrganisationRepo.GetByID(log,
		`select id,tenant_id,is_primary,legal_entity_name,organisation_type,facility_name,registration_no,license_number,license_expiry,gstin,address,data_sharing,status,created_at,updated_at from organisations where tenant_id=$1 and is_primary=true limit 1`,
		tenantID,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("organisation get primary failed",
				zap.String("tenant_id", tenantID),
				zap.String("reason", "not_found"),
			)
			return Organisation{}, wrapError.ErrOrganisationNotFound
		}
		log.Error("organisation get primary failed",
			zap.String("tenant_id", tenantID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return Organisation{}, wrapError.ErrOrganisationFetchFailed
	}
	return org, nil
}

func (s *OrganisationService) ListByTenantID(log *zap.Logger, tenantID string) ([]Organisation, error) {
	log = ensureLog(log)
	if strings.TrimSpace(tenantID) == "" {
		return nil, wrapError.ErrInvalidRequest
	}

	orgs, err := s.OrganisationRepo.ListByTenantID(log,
		`select id,tenant_id,is_primary,legal_entity_name,organisation_type,facility_name,registration_no,license_number,license_expiry,gstin,address,data_sharing,status,created_at,updated_at from organisations where tenant_id=$1 order by created_at desc`,
		tenantID,
	)
	if err != nil {
		log.Error("organisation list by tenant failed",
			zap.String("tenant_id", tenantID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, wrapError.ErrOrganisationFetchFailed
	}
	return orgs, nil
}

func (s *OrganisationService) UpdateAddress(log *zap.Logger, payload dto.OrganisationPayload) error {
	log = ensureLog(log)
	if strings.TrimSpace(payload.OrganisationID) == "" {
		return wrapError.ErrInvalidRequest
	}

	org := &Organisation{ID: payload.OrganisationID}
	org.Address = Address{
		CountryID: payload.CountryID,
		State:     payload.State,
		City:      payload.City,
		CreatedAt: time.Now(),
	}
	org.DataSharing = DataSharing{
		PatientLookup: payload.PatientLookup,
		LabReports:    payload.LabReports,
	}

	err := s.OrganisationRepo.UpdateAddressByID(log,
		`update organisations set address=$1, data_sharing=$2 where id=$3`,
		org.Address, org.DataSharing, org.ID,
	)
	if err != nil {
		return mapOrgUpdateErr(log, payload.OrganisationID, err)
	}

	log.Info("organisation address update success", zap.String("organisation_id", payload.OrganisationID))
	return nil
}

func (s *OrganisationService) Update(log *zap.Logger, organisationID string, payload dto.OrganisationPayload) error {
	log = ensureLog(log)
	if strings.TrimSpace(organisationID) == "" {
		return wrapError.ErrInvalidRequest
	}
	if err := validateUpdatePayload(payload); err != nil {
		return wrapError.ErrInvalidRequest
	}

	updateMap := buildUpdateMap(payload)
	if hasAddressFields(payload) {
		org, err := s.GetOrgByID(log, organisationID)
		if err != nil {
			return err
		}
		updateMap["address"] = mergeAddress(org.Address, payload)
	}
	if len(updateMap) == 1 {
		// only updated_at — nothing meaningful to change
		return wrapError.ErrInvalidRequest
	}

	query, args := buildOrganisationUpdateQuery(organisationID, updateMap)
	err := s.OrganisationRepo.Update(log, query, args...)
	if err != nil {
		return mapOrgUpdateErr(log, organisationID, err)
	}

	log.Info("organisation update success",
		zap.String("organisation_id", organisationID),
		zap.String("organisation_type", payload.OrganisationType),
	)
	return nil
}

func (s *OrganisationService) buildOrgModel(payload dto.OrganisationPayload) Organisation {
	status := strings.TrimSpace(payload.Status)
	if status == "" {
		status = constants.StatusActive
	}
	now := time.Now()
	return Organisation{
		ID:               uuid.NewString(),
		TenantID:         strings.TrimSpace(payload.TenantID),
		IsPrimary:        payload.IsPrimary,
		LegalEntityName:  strings.TrimSpace(payload.LegalEntityName),
		OrganisationType: strings.ToLower(strings.TrimSpace(payload.OrganisationType)),
		FacilityName:     strings.TrimSpace(payload.FacilityName),
		RegistrationNo:   strings.TrimSpace(payload.RegistrationNo),
		LicenseNumber:    strings.TrimSpace(payload.LicenseNumber),
		LicenseExpiry:    payload.LicenseExpiry,
		GSTIN:            strings.TrimSpace(payload.GSTIN),
		Address: Address{
			Address1:  strings.TrimSpace(payload.Address1),
			Address2:  strings.TrimSpace(payload.Address2),
			CountryID: strings.TrimSpace(payload.CountryID),
			State:     strings.TrimSpace(payload.State),
			City:      strings.TrimSpace(payload.City),
			CreatedAt: now,
		},
		DataSharing: DataSharing{
			PatientLookup: payload.PatientLookup,
			LabReports:    payload.LabReports,
		},
		Status:    strings.ToLower(status),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func validateAddPayload(payload dto.OrganisationPayload) error {
	required := []string{
		payload.TenantID,
		payload.LegalEntityName,
		payload.OrganisationType,
		payload.FacilityName,
	}
	for _, v := range required {
		if strings.TrimSpace(v) == "" {
			return wrapError.ErrInvalidRequest
		}
	}
	if !isAllowedOrganisationType(payload.OrganisationType) {
		return wrapError.ErrInvalidRequest
	}
	if status := strings.TrimSpace(payload.Status); status != "" && !isAllowedOrganisationStatus(status) {
		return wrapError.ErrInvalidRequest
	}
	return nil
}

func validateUpdatePayload(payload dto.OrganisationPayload) error {
	if typ := strings.TrimSpace(payload.OrganisationType); typ != "" && !isAllowedOrganisationType(typ) {
		return wrapError.ErrInvalidRequest
	}
	if status := strings.TrimSpace(payload.Status); status != "" && !isAllowedOrganisationStatus(status) {
		return wrapError.ErrInvalidRequest
	}
	return nil
}

func buildUpdateMap(payload dto.OrganisationPayload) map[string]interface{} {
	updateMap := map[string]interface{}{"updated_at": time.Now()}
	setIfNotEmpty(updateMap, "legal_entity_name", payload.LegalEntityName)
	if typ := strings.TrimSpace(payload.OrganisationType); typ != "" {
		updateMap["organisation_type"] = strings.ToLower(typ)
	}
	setIfNotEmpty(updateMap, "facility_name", payload.FacilityName)
	setIfNotEmpty(updateMap, "registration_no", payload.RegistrationNo)
	setIfNotEmpty(updateMap, "license_number", payload.LicenseNumber)
	setIfNotEmpty(updateMap, "gstin", payload.GSTIN)
	if status := strings.TrimSpace(payload.Status); status != "" {
		updateMap["status"] = strings.ToLower(status)
	}
	if payload.LicenseExpiry != nil {
		updateMap["license_expiry"] = payload.LicenseExpiry
	}
	return updateMap
}

func setIfNotEmpty(m map[string]interface{}, key, value string) {
	if strings.TrimSpace(value) != "" {
		m[key] = strings.TrimSpace(value)
	}
}

func hasAddressFields(payload dto.OrganisationPayload) bool {
	return strings.TrimSpace(payload.Address1) != "" ||
		strings.TrimSpace(payload.Address2) != "" ||
		strings.TrimSpace(payload.City) != "" ||
		strings.TrimSpace(payload.State) != "" ||
		strings.TrimSpace(payload.CountryID) != ""
}

func mergeAddress(existing Address, payload dto.OrganisationPayload) Address {
	if v := strings.TrimSpace(payload.Address1); v != "" {
		existing.Address1 = v
	}
	if v := strings.TrimSpace(payload.Address2); v != "" {
		existing.Address2 = v
	}
	if v := strings.TrimSpace(payload.City); v != "" {
		existing.City = v
	}
	if v := strings.TrimSpace(payload.State); v != "" {
		existing.State = v
	}
	if v := strings.TrimSpace(payload.CountryID); v != "" {
		existing.CountryID = v
	}
	return existing
}

func mapOrgUpdateErr(log *zap.Logger, organisationID string, err error) error {
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

// buildOrganisationUpdateQuery builds a parameterized UPDATE from a column map.
// Keys are sorted for stable SQL in tests.
func buildOrganisationUpdateQuery(organisationID string, update map[string]interface{}) (string, []any) {
	keys := make([]string, 0, len(update))
	for k := range update {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	setParts := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)+1)
	for i, key := range keys {
		setParts = append(setParts, fmt.Sprintf("%s=$%d", key, i+1))
		args = append(args, update[key])
	}
	query := fmt.Sprintf(
		"update organisations set %s where id=$%d",
		strings.Join(setParts, ", "),
		len(keys)+1,
	)
	args = append(args, organisationID)
	return query, args
}
