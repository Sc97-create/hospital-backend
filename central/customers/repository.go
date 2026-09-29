package customers

import (
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrCustomerNotFound = errors.New("customer not found")

type CustomerRepository interface {
	Create(log *zap.Logger, customer Customer) error
	ExistsByWorkEmail(log *zap.Logger, workEmail string) (bool, error)
	GetByID(log *zap.Logger, customerID string) (Customer, error)
	GetByTenantID(log *zap.Logger, tenantID string) (Customer, error)
	MarkEmailVerified(log *zap.Logger, customerID string, verifiedAt time.Time) error
	BindTenantID(log *zap.Logger, tx *gorm.DB, customerID, tenantID string) error

	CreateSignupCode(log *zap.Logger, code SignupCode) error
	FindActiveSignupCode(log *zap.Logger, userID, code string) (SignupCode, error)
	MarkSignupCodeVerified(log *zap.Logger, codeID string, verifiedAt time.Time) error
}

func (r *CustomerRepo) Create(log *zap.Logger, customer Customer) error {
	log = ensureLog(log)
	if err := r.db.Create(&customer).Error; err != nil {
		log.Error("customer repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (r *CustomerRepo) ExistsByWorkEmail(log *zap.Logger, workEmail string) (bool, error) {
	log = ensureLog(log)
	var count int64
	err := r.db.Model(&Customer{}).Where("work_email = ?", strings.ToLower(workEmail)).Count(&count).Error
	if err != nil {
		log.Error("customer repo error", zap.String("op", "ExistsByWorkEmail"), zap.Error(err))
		return false, err
	}
	return count > 0, nil
}

func (r *CustomerRepo) GetByID(log *zap.Logger, customerID string) (Customer, error) {
	log = ensureLog(log)
	var customer Customer
	err := r.db.Where("id = ?", customerID).First(&customer).Error
	if err != nil {
		log.Error("customer repo error", zap.String("op", "GetByID"), zap.Error(err))
		return Customer{}, err
	}
	return customer, nil
}

func (r *CustomerRepo) GetByTenantID(log *zap.Logger, tenantID string) (Customer, error) {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return Customer{}, ErrCustomerNotFound
	}
	var customer Customer
	err := r.db.Where("tenant_id = ?", tenantID).Order("created_at asc").First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Customer{}, ErrCustomerNotFound
		}
		log.Error("customer repo error", zap.String("op", "GetByTenantID"), zap.Error(err))
		return Customer{}, err
	}
	return customer, nil
}

func (r *CustomerRepo) MarkEmailVerified(log *zap.Logger, customerID string, verifiedAt time.Time) error {
	log = ensureLog(log)
	result := r.db.Model(&Customer{}).
		Where("id = ?", customerID).
		Updates(map[string]interface{}{
			"email_verified_at": verifiedAt,
			"updated_at":        verifiedAt,
		})
	if result.Error != nil {
		log.Error("customer repo error", zap.String("op", "MarkEmailVerified"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCustomerNotFound
	}
	return nil
}

// BindTenantID sets tenant_id on the customer. When tx is non-nil, the update runs in that transaction.
func (r *CustomerRepo) BindTenantID(log *zap.Logger, tx *gorm.DB, customerID, tenantID string) error {
	log = ensureLog(log)
	db := r.db
	if tx != nil {
		db = tx
	}
	result := db.Model(&Customer{}).Where("id = ?", customerID).Update("tenant_id", tenantID)
	if result.Error != nil {
		log.Error("customer repo error", zap.String("op", "BindTenantID"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCustomerNotFound
	}
	return nil
}

func (r *CustomerRepo) CreateSignupCode(log *zap.Logger, code SignupCode) error {
	log = ensureLog(log)
	if err := r.db.Create(&code).Error; err != nil {
		log.Error("customer repo error", zap.String("op", "CreateSignupCode"), zap.Error(err))
		return err
	}
	return nil
}

func (r *CustomerRepo) FindActiveSignupCode(log *zap.Logger, userID, code string) (SignupCode, error) {
	log = ensureLog(log)
	var row SignupCode
	err := r.db.
		Where("user_id = ? AND code = ? AND verified_at IS NULL", userID, code).
		Order("created_at desc").
		First(&row).Error
	if err != nil {
		log.Error("customer repo error", zap.String("op", "FindActiveSignupCode"), zap.Error(err))
		return SignupCode{}, err
	}
	return row, nil
}

func (r *CustomerRepo) MarkSignupCodeVerified(log *zap.Logger, codeID string, verifiedAt time.Time) error {
	log = ensureLog(log)
	result := r.db.Model(&SignupCode{}).
		Where("id = ?", codeID).
		Updates(map[string]interface{}{
			"verified_at": verifiedAt,
			"updated_at":  verifiedAt,
		})
	if result.Error != nil {
		log.Error("customer repo error", zap.String("op", "MarkSignupCodeVerified"), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCustomerNotFound
	}
	return nil
}
