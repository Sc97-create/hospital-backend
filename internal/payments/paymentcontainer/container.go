package paymentcontainer

import (
	"hospital-backend/config"
	"hospital-backend/internal/payments"
	"hospital-backend/internal/payments/module"

	"gorm.io/gorm"
)

type PaymentContainer struct {
	Mod *module.Module
}

func NewContainer(
	db *gorm.DB,
	cfg config.Config,
	prescriptionStatus payments.PrescriptionStatusUpdater,
	fulfillment payments.IPaymentFulfillment,
	subscriptionActivator payments.TenantEntitlementActivator,
) *PaymentContainer {
	mod := module.NewModule(db, cfg, prescriptionStatus, fulfillment, subscriptionActivator)
	return &PaymentContainer{
		Mod: mod,
	}
}
