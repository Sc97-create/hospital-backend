package jwt

import (
	"gorm.io/gorm"
)

type Refreshtoken struct {
	DB *gorm.DB
}

func NewRefreshRepo(db *gorm.DB) *Refreshtoken {
	return &Refreshtoken{DB: db}
}
