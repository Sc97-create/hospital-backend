package payments

import "gorm.io/gorm"

type DB struct {
	db *gorm.DB
}

func NewPaymentsDB(db *gorm.DB) *DB {
	return &DB{db: db}
}

type IPaymentsRepository interface {
	Create(db *gorm.DB, payment Payments) (err error)
}

func (c *DB) Create(db *gorm.DB, payments Payments) (err error) {
	return db.Create(&payments).Error
}
