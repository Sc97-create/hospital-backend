package paymentcontainer

import (
	"hospital-backend/config"
	"hospital-backend/internal/payments/module"

	"gorm.io/gorm"
)

type PaymentContainer struct {
	Mod *module.Module
}

func NewContainer(db *gorm.DB, cfg config.Config) *PaymentContainer {
	mod := module.NewModule(db, cfg)
	return &PaymentContainer{
		Mod: mod,
	}

}
