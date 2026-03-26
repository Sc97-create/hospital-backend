package db

import (
	"gorm.io/gorm"
)

type Postgre struct {
	GormDriver *gorm.DB
}
