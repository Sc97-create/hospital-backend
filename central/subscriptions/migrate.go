package subscriptions

import "gorm.io/gorm"

// AutoMigrate creates the subscriptions table.
// Order and signature columns belong on payments, so any leftover copies are dropped.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Subscription{}); err != nil {
		return err
	}
	return db.Exec(`
		ALTER TABLE subscriptions
		  DROP COLUMN IF EXISTS order_id,
		  DROP COLUMN IF EXISTS provider_payment_id,
		  DROP COLUMN IF EXISTS payment_signature
	`).Error
}
