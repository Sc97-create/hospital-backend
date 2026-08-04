package payments

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IPaymentAttempts interface {
	CreatePaymentAttempts(log *zap.Logger, tx *gorm.DB, PAttempts PaymentAttempts) error
	FindByProviderLinkID(log *zap.Logger, providerLinkID string) (PaymentAttempts, error)
	FindByPaymentID(log *zap.Logger, paymentID string) (PaymentAttempts, error)
	CountByPaymentID(log *zap.Logger, paymentID string) (int64, error)
	FindByClientIdempotencyKey(log *zap.Logger, key string) (PaymentAttempts, error)
	UpdatePaymentAttempt(log *zap.Logger, tx *gorm.DB, paymentAttempt PaymentAttempts) error
	UpdatePaymentAttemptStatus(log *zap.Logger, tx *gorm.DB, PaymentAttempt PaymentAttempts) error
	// ClaimForProcessing atomically transitions status from pending → processing.
	// Returns (attempt, true, nil) if claimed, (attempt, false, nil) if already claimed by another goroutine.
	ClaimForProcessing(log *zap.Logger, providerLinkID string) (PaymentAttempts, bool, error)
}

func (p *DB) CreatePaymentAttempts(log *zap.Logger, tx *gorm.DB, PAttempts PaymentAttempts) error {
	log = ensureLog(log)
	err := tx.Create(&PAttempts).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "CreatePaymentAttempts"), zap.Error(err))
		return err
	}
	return nil
}

func (p *DB) FindByProviderLinkID(log *zap.Logger, providerLinkID string) (PaymentAttempts, error) {
	log = ensureLog(log)
	var attempt PaymentAttempts
	err := p.db.Where("provider_link_id = ?", providerLinkID).First(&attempt).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("payments repo error", zap.String("op", "FindByProviderLinkID"), zap.Error(err))
		}
		return attempt, err
	}
	return attempt, nil
}

func (p *DB) FindByPaymentID(log *zap.Logger, paymentID string) (PaymentAttempts, error) {
	log = ensureLog(log)
	var attempt PaymentAttempts
	err := p.db.Where("payment_id = ?", paymentID).Order("created_at DESC").First(&attempt).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("payments repo error", zap.String("op", "FindByPaymentID"), zap.Error(err))
		}
		return attempt, err
	}
	return attempt, nil
}

func (p *DB) CountByPaymentID(log *zap.Logger, paymentID string) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := p.db.Model(&PaymentAttempts{}).Where("payment_id = ?", paymentID).Count(&count).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "CountByPaymentID"), zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (p *DB) FindByClientIdempotencyKey(log *zap.Logger, key string) (PaymentAttempts, error) {
	log = ensureLog(log)
	var attempt PaymentAttempts
	err := p.db.Where("provider_request->>'client_idempotency_key' = ?", key).
		Order("created_at DESC").
		First(&attempt).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("payments repo error", zap.String("op", "FindByClientIdempotencyKey"), zap.Error(err))
		}
		return attempt, err
	}
	return attempt, nil
}

func (p *DB) UpdatePaymentAttempt(log *zap.Logger, tx *gorm.DB, paymentAttempt PaymentAttempts) error {
	log = ensureLog(log)
	err := tx.Model(&PaymentAttempts{}).Where("id = ?", paymentAttempt.ID).Updates(paymentAttempt).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "UpdatePaymentAttempt"), zap.Error(err))
		return err
	}
	return nil
}

func (p *DB) UpdatePaymentAttemptStatus(log *zap.Logger, tx *gorm.DB, paymentAttempt PaymentAttempts) error {
	log = ensureLog(log)
	err := tx.Model(&PaymentAttempts{}).Where("id = ?", paymentAttempt.ID).Updates(paymentAttempt).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "UpdatePaymentAttemptStatus"), zap.Error(err))
		return err
	}
	return nil
}

func (p *DB) ClaimForProcessing(log *zap.Logger, providerLinkID string) (PaymentAttempts, bool, error) {
	log = ensureLog(log)
	result := p.db.Model(&PaymentAttempts{}).
		Where("provider_link_id = ? AND payment_status = ?", providerLinkID, "pending").
		Update("payment_status", "processing")
	if result.Error != nil {
		log.Error("payments repo error", zap.String("op", "ClaimForProcessing"), zap.Error(result.Error))
		return PaymentAttempts{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return PaymentAttempts{}, false, nil
	}
	var attempt PaymentAttempts
	err := p.db.Where("provider_link_id = ?", providerLinkID).First(&attempt).Error
	if err != nil {
		log.Error("payments repo error", zap.String("op", "ClaimForProcessing"), zap.Error(err))
		return PaymentAttempts{}, false, err
	}
	return attempt, true, nil
}
