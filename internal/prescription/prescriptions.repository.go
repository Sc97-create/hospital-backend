package prescription

import (
	"hospital-backend/internal/prescription/dto"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PrescriptionDB struct {
	db *gorm.DB
}

func NewPrescriptionDB(db *gorm.DB) *PrescriptionDB {
	return &PrescriptionDB{db: db}
}

type PrescriptionRepositoryInterface interface {
	CreatePrescription(log *zap.Logger, db *gorm.DB, prescription Prescription) error
	GetPrescriptionByID(log *zap.Logger, id string) (*Prescription, error)
	GetPrescriptionsByAppointmentID(log *zap.Logger, query string, cond ...any) ([]PrescriptionAppointmentData, error)
	GetPrescriptionByAppointmentIDCount(log *zap.Logger, cond ...any) (count int64, err error)
	GetPrescriptionsByPatientID(log *zap.Logger, query string, args ...any) ([]dto.PrescriptionListItem, error)
	GetPrescriptionByPatientIDCount(log *zap.Logger, patientID string) (count int64, err error)
	GetPrescriptionsByDoctorID(log *zap.Logger, doctorID string) ([]Prescription, error)
	DeletePrescription(log *zap.Logger, id string) error
	FindMany(log *zap.Logger, query string, args ...any) ([]dto.PrescriptionListItem, error)
	FindByStatus(log *zap.Logger, organisationID string, status string, limit int, offset int) ([]dto.PrescriptionListItem, error)
	CountByStatus(log *zap.Logger, organisationID string, status string) (int64, error)
	FindPrescriptionByID(log *zap.Logger, query string, id string) (presc Prescription, err error)
	UpdateStatus(log *zap.Logger, tx *gorm.DB, status string, prescriptionID string) (err error)
	GetNotificationDetails(log *zap.Logger, query string, prescriptionID string) (PrescriptionNotificationData, error)
	Count(log *zap.Logger, query string, args ...any) (int64, error)
}

func (pdb *PrescriptionDB) CreatePrescription(log *zap.Logger, db *gorm.DB, prescription Prescription) error {
	log = ensureLog(log)
	err := db.Create(&prescription).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "CreatePrescription"), zap.Error(err))
		return err
	}
	return nil
}

func (pdb *PrescriptionDB) GetPrescriptionByID(log *zap.Logger, id string) (*Prescription, error) {
	log = ensureLog(log)
	var prescription Prescription
	err := pdb.db.First(&prescription, "id = ?", id).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Error("prescription repo error", zap.String("op", "GetPrescriptionByID"), zap.Error(err))
		}
		return &prescription, err
	}
	return &prescription, err
}

func (pdb *PrescriptionDB) GetPrescriptionsByAppointmentID(log *zap.Logger, query string, cond ...any) ([]PrescriptionAppointmentData, error) {
	log = ensureLog(log)
	var prescriptions []PrescriptionAppointmentData
	err := pdb.db.Raw(query, cond...).Find(&prescriptions).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "GetPrescriptionsByAppointmentID"), zap.Error(err))
		return nil, err
	}
	return prescriptions, nil
}

func (pdb *PrescriptionDB) GetPrescriptionsByPatientID(log *zap.Logger, query string, args ...any) ([]dto.PrescriptionListItem, error) {
	log = ensureLog(log)
	var prescriptions []dto.PrescriptionListItem
	if err := pdb.db.Raw(query, args...).Find(&prescriptions).Error; err != nil {
		log.Error("prescription repo error", zap.String("op", "GetPrescriptionsByPatientID"), zap.Error(err))
		return nil, err
	}
	return prescriptions, nil
}

func (pdb *PrescriptionDB) GetPrescriptionByPatientIDCount(log *zap.Logger, patientID string) (count int64, err error) {
	log = ensureLog(log)
	err = pdb.db.Model(&Prescription{}).
		Where("patient_id = ?", patientID).
		Count(&count).
		Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "GetPrescriptionByPatientIDCount"), zap.Error(err))
	}
	return count, err
}

func (pdb *PrescriptionDB) GetPrescriptionsByDoctorID(log *zap.Logger, doctorID string) ([]Prescription, error) {
	log = ensureLog(log)
	var prescriptions []Prescription
	err := pdb.db.Where("prescribed_by = ?", doctorID).Find(&prescriptions).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "GetPrescriptionsByDoctorID"), zap.Error(err))
		return nil, err
	}
	return prescriptions, err
}

func (pdb *PrescriptionDB) DeletePrescription(log *zap.Logger, id string) error {
	log = ensureLog(log)
	err := pdb.db.Delete(&Prescription{}, "id = ?", id).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "DeletePrescription"), zap.Error(err))
		return err
	}
	return nil
}

func (pdb *PrescriptionDB) FindPrescriptionByID(log *zap.Logger, query string, id string) (presc Prescription, err error) {
	log = ensureLog(log)
	err = pdb.db.Raw(query, id).Scan(&presc).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "FindPrescriptionByID"), zap.Error(err))
		return
	}
	return
}

func (pdb *PrescriptionDB) FindMany(log *zap.Logger, query string, args ...any) (prescription []dto.PrescriptionListItem, err error) {
	log = ensureLog(log)
	err = pdb.db.Raw(query, args...).Scan(&prescription).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "FindMany"), zap.Error(err))
		return
	}
	return
}

func (pdb *PrescriptionDB) FindByStatus(log *zap.Logger, organisationID string, status string, limit int, offset int) ([]dto.PrescriptionListItem, error) {
	log = ensureLog(log)
	var prescriptions []dto.PrescriptionListItem
	err := pdb.db.Table("prescriptions AS p").
		Select("p.id, p.code, e.username AS prescribed_by, p.patient_id, pt.name AS patient_name, p.appointment_id, p.created_at, p.status").
		Joins("JOIN users AS e ON p.prescribed_by = e.id").
		Joins("JOIN patients AS pt ON p.patient_id = pt.id").
		Where("p.organisation_id = ? AND p.status = ?", organisationID, status).
		Order("p.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&prescriptions).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "FindByStatus"), zap.Error(err))
		return nil, err
	}
	return prescriptions, nil
}

func (pdb *PrescriptionDB) CountByStatus(log *zap.Logger, organisationID string, status string) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := pdb.db.Model(&Prescription{}).
		Where("organisation_id = ? AND status = ?", organisationID, status).
		Count(&count).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "CountByStatus"), zap.Error(err))
		return count, err
	}
	return count, err
}

func (pdb *PrescriptionDB) UpdateStatus(log *zap.Logger, db *gorm.DB, status string, prescriptionID string) (err error) {
	log = ensureLog(log)
	query := `UPDATE prescriptions
	SET status = ?, updated_at = ?
	WHERE id = ?;`
	err = db.Exec(query, status, time.Now(), prescriptionID).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "UpdateStatus"), zap.Error(err))
		return
	}
	return
}

func (pdb *PrescriptionDB) Count(log *zap.Logger, query string, args ...any) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := pdb.db.Raw(query, args...).Scan(&count).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "Count"), zap.Error(err))
		return count, err
	}
	return count, err
}

func (pdb *PrescriptionDB) GetPrescriptionByAppointmentIDCount(log *zap.Logger, cond ...any) (count int64, err error) {
	log = ensureLog(log)
	err = pdb.db.
		Model(&Prescription{}).
		Where("appointment_id = ? AND organisation_id = ?", cond...).
		Count(&count).
		Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "GetPrescriptionByAppointmentIDCount"), zap.Error(err))
		return
	}
	return
}

func (pdb *PrescriptionDB) GetNotificationDetails(log *zap.Logger, query string, prescriptionID string) (PrescriptionNotificationData, error) {
	log = ensureLog(log)
	var data PrescriptionNotificationData
	err := pdb.db.Raw(query, prescriptionID).Scan(&data).Error
	if err != nil {
		log.Error("prescription repo error", zap.String("op", "GetNotificationDetails"), zap.Error(err))
		return PrescriptionNotificationData{}, err
	}
	return data, nil
}
