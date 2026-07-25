package payments

import "gorm.io/gorm"

type IPaymentAttempts interface {
	CreatePaymentAttempts(tx *gorm.DB, PAttempts PaymentAttempts) error
	FindByProviderLinkID(providerLinkID string) (PaymentAttempts, error)
	FindByPaymentID(paymentID string) (PaymentAttempts, error)
	CountByPaymentID(paymentID string) (int64, error)
	FindByClientIdempotencyKey(key string) (PaymentAttempts, error)
	UpdatePaymentAttempt(tx *gorm.DB, paymentAttempt PaymentAttempts) error
	UpdatePaymentAttemptStatus(tx *gorm.DB, PaymentAttempt PaymentAttempts) error
	// ClaimForProcessing atomically transitions status from pending → processing.
	// Returns (attempt, true, nil) if claimed, (attempt, false, nil) if already claimed by another goroutine.
	ClaimForProcessing(providerLinkID string) (PaymentAttempts, bool, error)
}

func (p *DB) CreatePaymentAttempts(tx *gorm.DB, PAttempts PaymentAttempts) error {
	return tx.Create(&PAttempts).Error
}

func (p *DB) FindByProviderLinkID(providerLinkID string) (PaymentAttempts, error) {
	var attempt PaymentAttempts
	err := p.db.Where("provider_link_id = ?", providerLinkID).First(&attempt).Error
	return attempt, err
}

func (p *DB) FindByPaymentID(paymentID string) (PaymentAttempts, error) {
	var attempt PaymentAttempts
	err := p.db.Where("payment_id = ?", paymentID).Order("created_at DESC").First(&attempt).Error
	return attempt, err
}

func (p *DB) CountByPaymentID(paymentID string) (int64, error) {
	var count int64
	err := p.db.Model(&PaymentAttempts{}).Where("payment_id = ?", paymentID).Count(&count).Error
	return count, err
}

func (p *DB) FindByClientIdempotencyKey(key string) (PaymentAttempts, error) {
	var attempt PaymentAttempts
	err := p.db.Where("provider_request->>'client_idempotency_key' = ?", key).
		Order("created_at DESC").
		First(&attempt).Error
	return attempt, err
}

func (p *DB) UpdatePaymentAttempt(tx *gorm.DB, paymentAttempt PaymentAttempts) error {
	return tx.Model(&PaymentAttempts{}).Where("id = ?", paymentAttempt.ID).Updates(paymentAttempt).Error
}

func (p *DB) UpdatePaymentAttemptStatus(tx *gorm.DB, paymentAttempt PaymentAttempts) error {
	return tx.Model(&PaymentAttempts{}).Where("id = ?", paymentAttempt.ID).Updates(paymentAttempt).Error
}

// ClaimForProcessing atomically marks the attempt as processing so only one concurrent
// webhook call proceeds. Any duplicate webhook returns claimed=false and is safely ignored.
func (p *DB) ClaimForProcessing(providerLinkID string) (PaymentAttempts, bool, error) {
	result := p.db.Model(&PaymentAttempts{}).
		Where("provider_link_id = ? AND payment_status = ?", providerLinkID, "pending").
		Update("payment_status", "processing")
	if result.Error != nil {
		return PaymentAttempts{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		// already claimed or processed by another goroutine
		return PaymentAttempts{}, false, nil
	}
	var attempt PaymentAttempts
	err := p.db.Where("provider_link_id = ?", providerLinkID).First(&attempt).Error
	return attempt, true, err
}
