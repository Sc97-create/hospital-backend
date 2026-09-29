package tenants

import (
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TenantRepository interface {
	Create(log *zap.Logger, tx *gorm.DB, tenant Tenant) error
	GetByID(log *zap.Logger, tenantID string) (Tenant, error)
	UpdateStatusByID(log *zap.Logger, tx *gorm.DB, tenantID, status string) error
	UpdateByID(log *zap.Logger, tx *gorm.DB, tenantID string, updates map[string]interface{}) error
}

func (r *TenantRepo) Create(log *zap.Logger, tx *gorm.DB, tenant Tenant) error {
	log = ensureLog(log)
	db := r.db
	if tx != nil {
		db = tx
	}
	if err := db.Create(&tenant).Error; err != nil {
		log.Error("tenant repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (r *TenantRepo) GetByID(log *zap.Logger, tenantID string) (Tenant, error) {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return Tenant{}, gorm.ErrInvalidData
	}
	var tenant Tenant
	err := r.db.Where("id = ?", tenantID).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Tenant{}, err
		}
		log.Error("tenant repo error", zap.String("op", "GetByID"), zap.Error(err))
		return Tenant{}, err
	}
	return tenant, nil
}

func (r *TenantRepo) UpdateStatusByID(log *zap.Logger, tx *gorm.DB, tenantID, status string) error {
	status = strings.TrimSpace(status)
	if status == "" {
		return gorm.ErrInvalidData
	}
	return r.UpdateByID(log, tx, tenantID, map[string]interface{}{"status": status})
}

func (r *TenantRepo) UpdateByID(log *zap.Logger, tx *gorm.DB, tenantID string, updates map[string]interface{}) error {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" || len(updates) == 0 {
		return gorm.ErrInvalidData
	}
	db := r.db
	if tx != nil {
		db = tx
	}
	updates["updated_at"] = time.Now()
	result := db.Model(&Tenant{}).Where("id = ?", tenantID).Updates(updates)
	if result.Error != nil {
		log.Error("tenant repo error", zap.String("op", "UpdateByID"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
