package diagnosticcentre

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DiagnosticCentreRepository persists diagnostic centre rows.
// List/count/update SQL is owned by the service and executed here via Raw/Exec.
type DiagnosticCentreRepository interface {
	Create(log *zap.Logger, centre *DiagnosticCentre) error
	FindMany(log *zap.Logger, query string, args ...any) ([]DiagnosticCentre, error)
	Count(log *zap.Logger, query string, args ...any) (int64, error)
	Exec(log *zap.Logger, query string, args ...any) error
}

type diagnosticCentreDB struct {
	db *gorm.DB
}

func NewDiagnosticCentreRepo(db *gorm.DB) DiagnosticCentreRepository {
	return &diagnosticCentreDB{db: db}
}

func (r *diagnosticCentreDB) Create(log *zap.Logger, centre *DiagnosticCentre) error {
	log = ensureLog(log)
	if err := r.db.Create(centre).Error; err != nil {
		log.Error("diagnostic centre repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (r *diagnosticCentreDB) FindMany(log *zap.Logger, query string, args ...any) ([]DiagnosticCentre, error) {
	log = ensureLog(log)
	var rows []DiagnosticCentre
	if err := r.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		log.Error("diagnostic centre repo error", zap.String("op", "FindMany"), zap.Error(err))
		return nil, err
	}
	if rows == nil {
		rows = []DiagnosticCentre{}
	}
	return rows, nil
}

func (r *diagnosticCentreDB) Count(log *zap.Logger, query string, args ...any) (int64, error) {
	log = ensureLog(log)
	var count int64
	if err := r.db.Raw(query, args...).Scan(&count).Error; err != nil {
		log.Error("diagnostic centre repo error", zap.String("op", "Count"), zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (r *diagnosticCentreDB) Exec(log *zap.Logger, query string, args ...any) error {
	log = ensureLog(log)
	res := r.db.Exec(query, args...)
	if res.Error != nil {
		log.Error("diagnostic centre repo error", zap.String("op", "Exec"), zap.Error(res.Error))
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
