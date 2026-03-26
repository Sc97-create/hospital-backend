package organisation

import (
	"gorm.io/gorm"
)

type OrganisationRepo interface {
	Create(*gorm.DB, *Organisation) error
	GetOrganisationByID(organisationID string) (*Organisation, error)
	UpdateLocationByID(organisation *Organisation) error
	Update(organisationID string, update map[string]interface{}) error
}

func (ORepo *OrgRepo) Create(tx *gorm.DB, organisation *Organisation) error {
	err := tx.Create(&organisation).Error
	if err != nil {
		return tx.Error
	}
	return nil
}
func (ORepo *OrgRepo) GetOrganisationByID(organisationID string) (*Organisation, error) {
	orgModel := Organisation{}
	query := `select id,organisation_name,code,legal_entity_name,hospital_type,address,security from organisations where id=$1`
	err := ORepo.db.Raw(query, organisationID).Scan(&orgModel).Error
	if err != nil {
		return nil, err
	}
	return &orgModel, nil
}
func (ORepo *OrgRepo) UpdateLocationByID(organisation *Organisation) error {
	orgModel := Organisation{}
	err := ORepo.db.Model(&orgModel).Where("id=?", organisation.ID).Updates(
		map[string]interface{}{
			"address":  organisation.Address,
			"security": organisation.Security,
		}).Error
	if err != nil {
		return err
	}
	return nil
}
func (ORepo *OrgRepo) Update(organisationID string, update map[string]interface{}) error {
	orgModel := Organisation{}
	err := ORepo.db.Model(&orgModel).Where("id=?", organisationID).Updates(update).Error
	if err != nil {
		return err
	}
	return nil
}
