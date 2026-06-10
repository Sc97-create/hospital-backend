package admins

import "gorm.io/gorm"

type CommonDB struct {
	db *gorm.DB
}

func NewCommonDB(db *gorm.DB) *CommonDB {
	return &CommonDB{db: db}
}
