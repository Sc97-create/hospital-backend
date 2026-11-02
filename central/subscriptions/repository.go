package subscriptions

import (
	"errors"
	"strings"
	"time"

	"hospital-backend/central/plans"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

type SubscriptionRepository interface {
	Create(log *zap.Logger, tx *gorm.DB, sub Subscription) error
	GetByID(log *zap.Logger, subscriptionID string) (Subscription, error)
	GetLatestByTenantID(log *zap.Logger, tenantID string) (Subscription, error)
	GetLatestWithPlanByTenantID(log *zap.Logger, tenantID string) (SubscriptionPlanRow, error)
	UpdateByID(log *zap.Logger, tx *gorm.DB, id string, updates map[string]interface{}) error
}

// SubscriptionPlanRow is the latest subscription for a tenant joined to its plan.
type SubscriptionPlanRow struct {
	ID           string
	TenantID     string
	PlanID       string
	Status       string
	BillingCycle string
	Price        float64
	StartAt      *time.Time
	EndAt        *time.Time
	PlanName     string
	PlanStatus   string
	MonthlyPrice float64
	Discount     float64
	PlanDetails  plans.PlanDetails
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

func (r *SubscriptionRepo) GetByID(log *zap.Logger, subscriptionID string) (Subscription, error) {
	log = ensureLog(log)
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return Subscription{}, ErrSubscriptionNotFound
	}
	var sub Subscription
	err := r.db.Where("id = ?", subscriptionID).First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Subscription{}, ErrSubscriptionNotFound
		}
		log.Error("subscription repo error", zap.String("op", "GetByID"), zap.Error(err))
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

func (r *SubscriptionRepo) GetLatestWithPlanByTenantID(log *zap.Logger, tenantID string) (SubscriptionPlanRow, error) {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return SubscriptionPlanRow{}, ErrSubscriptionNotFound
	}
	var row SubscriptionPlanRow
	err := r.db.Raw(`
		SELECT
			s.id,
			s.tenant_id,
			s.plan_id,
			s.status,
			s.billing_cycle,
			s.price,
			s.start_at,
			s.end_at,
			p.name AS plan_name,
			p.status AS plan_status,
			p.monthly_price,
			p.discount,
			p.plan_details
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.tenant_id = ?
		ORDER BY s.created_at DESC
		LIMIT 1
	`, tenantID).Scan(&row).Error
	if err != nil {
		log.Error("subscription repo error", zap.String("op", "GetLatestWithPlanByTenantID"), zap.Error(err))
		return SubscriptionPlanRow{}, err
	}
	if row.ID == "" {
		return SubscriptionPlanRow{}, ErrSubscriptionNotFound
	}
	return row, nil
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
