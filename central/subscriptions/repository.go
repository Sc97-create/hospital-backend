package subscriptions

import (
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

type SubscriptionRepository interface {
	Create(log *zap.Logger, tx *gorm.DB, sub Subscription) error
	GetByOrderID(log *zap.Logger, orderID string) (Subscription, error)
	GetLatestByTenantID(log *zap.Logger, tenantID string) (Subscription, error)
	UpdateByID(log *zap.Logger, tx *gorm.DB, id string, updates map[string]interface{}) error
}

func (r *SubscriptionRepo) Create(log *zap.Logger, tx *gorm.DB, sub Subscription) error {
	log = ensureLog(log)
	db := r.db
	if tx != nil {
		db = tx
	}
	if err := db.Create(&sub).Error; err != nil {
		log.Error("subscription repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (r *SubscriptionRepo) GetByOrderID(log *zap.Logger, orderID string) (Subscription, error) {
	log = ensureLog(log)
	var sub Subscription
	err := r.db.Where("order_id = ?", orderID).First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Subscription{}, ErrSubscriptionNotFound
		}
		log.Error("subscription repo error", zap.String("op", "GetByOrderID"), zap.Error(err))
		return Subscription{}, err
	}
	return sub, nil
}

func (r *SubscriptionRepo) GetLatestByTenantID(log *zap.Logger, tenantID string) (Subscription, error) {
	log = ensureLog(log)
	var sub Subscription
	err := r.db.Where("tenant_id = ?", strings.TrimSpace(tenantID)).Order("created_at desc").First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Subscription{}, ErrSubscriptionNotFound
		}
		log.Error("subscription repo error", zap.String("op", "GetLatestByTenantID"), zap.Error(err))
		return Subscription{}, err
	}
	return sub, nil
}

func (r *SubscriptionRepo) UpdateByID(log *zap.Logger, tx *gorm.DB, id string, updates map[string]interface{}) error {
	log = ensureLog(log)
	db := r.db
	if tx != nil {
		db = tx
	}
	updates["updated_at"] = time.Now()
	result := db.Model(&Subscription{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		log.Error("subscription repo error", zap.String("op", "UpdateByID"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}
