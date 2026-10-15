package labservice

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LabOrderRepository persists lab order rows.
// List/count/detail SQL is owned by the service and executed here via Raw/Exec.
type LabOrderRepository interface {
	Create(log *zap.Logger, order *LabOrders) error
	FindByID(log *zap.Logger, id string) (LabOrders, error)
	FindMany(log *zap.Logger, query string, args ...any) ([]LabOrderListRow, error)
	Count(log *zap.Logger, query string, args ...any) (int64, error)
	Save(log *zap.Logger, order *LabOrders) error
	Exec(log *zap.Logger, query string, args ...any) error
}

type labOrderDB struct {
	db *gorm.DB
}

func NewLabOrderRepo(db *gorm.DB) LabOrderRepository {
	return &labOrderDB{db: db}
}

func (r *labOrderDB) Create(log *zap.Logger, order *LabOrders) error {
	log = ensureLog(log)
	if err := r.db.Create(order).Error; err != nil {
		log.Error("lab order repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (r *labOrderDB) FindByID(log *zap.Logger, id string) (LabOrders, error) {
	log = ensureLog(log)
	var order LabOrders
	err := r.db.Model(&LabOrders{}).Where("id = ?", id).First(&order).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("lab order repo error", zap.String("op", "FindByID"), zap.Error(err))
		}
		return LabOrders{}, err
	}
	return order, nil
}

func (r *labOrderDB) FindMany(log *zap.Logger, query string, args ...any) ([]LabOrderListRow, error) {
	log = ensureLog(log)
	var rows []LabOrderListRow
	if err := r.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		log.Error("lab order repo error", zap.String("op", "FindMany"), zap.Error(err))
		return nil, err
	}
	if rows == nil {
		rows = []LabOrderListRow{}
	}
	return rows, nil
}

func (r *labOrderDB) Count(log *zap.Logger, query string, args ...any) (int64, error) {
	log = ensureLog(log)
	var count int64
	if err := r.db.Raw(query, args...).Scan(&count).Error; err != nil {
		log.Error("lab order repo error", zap.String("op", "Count"), zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (r *labOrderDB) Save(log *zap.Logger, order *LabOrders) error {
	log = ensureLog(log)
	if err := r.db.Save(order).Error; err != nil {
		log.Error("lab order repo error", zap.String("op", "Save"), zap.Error(err))
		return err
	}
	return nil
}

func (r *labOrderDB) Exec(log *zap.Logger, query string, args ...any) error {
	log = ensureLog(log)
	res := r.db.Exec(query, args...)
	if res.Error != nil {
		log.Error("lab order repo error", zap.String("op", "Exec"), zap.Error(res.Error))
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
