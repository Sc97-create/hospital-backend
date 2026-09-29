package subscriptions

import "gorm.io/gorm"

// AutoMigrate creates the subscriptions table.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Subscription{})
}
