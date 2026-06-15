package repository

import "gorm.io/gorm"

type Db struct {
	DB *gorm.DB
}

func NewDB(db *gorm.DB) Db {
	return Db{DB: db}
}
