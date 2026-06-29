package payments

import "gorm.io/gorm"

type IPaymentAttempts interface {
	CreatePaymentAttempts(tx *gorm.DB, PAttempts PaymentAttempts) error
}

func (p *DB) CreatePaymentAttempts(tx *gorm.DB, PAttempts PaymentAttempts) error {
	return tx.Create(&PAttempts).Error
}
