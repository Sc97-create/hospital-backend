package notifications

import "gorm.io/gorm"

func RelaxRecipientConstraints(db *gorm.DB) error {
	return db.Exec(`ALTER TABLE notifications ALTER COLUMN patient_id DROP NOT NULL`).Error
}
