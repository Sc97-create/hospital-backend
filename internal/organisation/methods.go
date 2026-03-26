package organisation

import (
	"gorm.io/gorm"
)

type OrgRepo struct {
	db *gorm.DB
}

func NewOrganisationRepo(db *gorm.DB) *OrgRepo {
	return &OrgRepo{db: db}
}
