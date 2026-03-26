package organisation

import (
	"hospital-backend/internal/department"
	"hospital-backend/internal/license"
	dto "hospital-backend/internal/organisation/DTO"
	"hospital-backend/internal/roles"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// service should not depend on internal repo field
type OrganisationService struct {
	DB               *gorm.DB
	OrganisationRepo OrganisationRepo
	LicenseRep       *license.LicenseService
}

func NewOrganisationService(db *gorm.DB, orgRepo OrganisationRepo, license *license.LicenseService) *OrganisationService {
	return &OrganisationService{DB: db, OrganisationRepo: orgRepo, LicenseRep: license}
}

func (OService *OrganisationService) CreateWithLicense(payloadRequest dto.OrganisationPayload) (ID string, departmentID string, roleID string, err error) {
	organisation := new(Organisation)
	departmentModel := department.Department{}
	roleModel := roles.Role{}
	organisation.ID = uuid.NewString()
	organisation.OrganisationName = payloadRequest.OrganisationName
	organisation.LegalEntityName = payloadRequest.LegalEntityName
	organisation.HospitalType = payloadRequest.HospitalType
	organisation.CreatedAt = time.Now()
	organisation.UpdatedAt = time.Now()
	organisation.Code = uuid.NewString()
	err = OService.DB.Transaction(func(tx *gorm.DB) error {
		err := OService.OrganisationRepo.Create(tx, organisation)
		if err != nil {
			return err
		}
		err = OService.LicenseRep.CreateLicenseSrv(tx, organisation.OrganisationName, 6, organisation.ID, "month", time.Now())
		if err != nil {
			return err
		}
		return nil
	})

	return organisation.ID, departmentModel.ID, roleModel.ID, nil
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
func (Oservice *OrganisationService) GetByID(organisationID string) (*Organisation, error) {
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
