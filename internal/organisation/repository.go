package organisation

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrganisationRepo interface {
	Create(log *zap.Logger, tx *gorm.DB, organisation Organisation) error
	GetOrganisationByID(log *zap.Logger, organisationID string) (Organisation, error)
	UpdateLocationByID(log *zap.Logger, organisation *Organisation) error
	Update(log *zap.Logger, organisationID string, update map[string]interface{}) error
}

func (ORepo *OrgRepo) Create(log *zap.Logger, tx *gorm.DB, organisation Organisation) error {
	log = ensureLog(log)
	err := tx.Create(&organisation).Error
	if err != nil {
		log.Error("organisation repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (ORepo *OrgRepo) GetOrganisationByID(log *zap.Logger, organisationID string) (Organisation, error) {
	log = ensureLog(log)
	var orgModel Organisation
	query := `select id,organisation_name,code,legal_entity_name,hospital_type,address,security from organisations where id=$1`
	err := ORepo.db.Raw(query, organisationID).Scan(&orgModel).Error
	if err != nil {
		log.Error("organisation repo error", zap.String("op", "GetOrganisationByID"), zap.Error(err))
		return Organisation{}, err
	}
	// Raw().Scan() returns nil error with zero rows — treat empty ID as not found.
	if orgModel.ID == "" {
		return Organisation{}, gorm.ErrRecordNotFound
	}
	return orgModel, nil
}

func (ORepo *OrgRepo) UpdateLocationByID(log *zap.Logger, organisation *Organisation) error {
	log = ensureLog(log)
	result := ORepo.db.Model(&Organisation{}).Where("id=?", organisation.ID).Updates(
		map[string]interface{}{
			"address":  organisation.Address,
			"security": organisation.Security,
		})
	if result.Error != nil {
		log.Error("organisation repo error", zap.String("op", "UpdateLocationByID"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (ORepo *OrgRepo) Update(log *zap.Logger, organisationID string, update map[string]interface{}) error {
	log = ensureLog(log)
	result := ORepo.db.Model(&Organisation{}).Where("id=?", organisationID).Updates(update)
	if result.Error != nil {
		log.Error("organisation repo error", zap.String("op", "Update"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
