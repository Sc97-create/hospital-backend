package prescription

import (
	"hospital-backend/internal/prescription/dto"
	"time"

	"gorm.io/gorm"
)

type PrescriptionDB struct {
	db *gorm.DB
}

func NewPrescriptionDB(db *gorm.DB) *PrescriptionDB {
	return &PrescriptionDB{db: db}
}

type PrescriptionRepositoryInterface interface {
	CreatePrescription(db *gorm.DB, prescription Prescription) error
	GetPrescriptionByID(id string) (*Prescription, error)
	GetPrescriptionsByAppointmentID(query string, cond ...any) ([]PrescriptionAppointmentData, error)
	GetPrescriptionByAppointmentIDCount(cond ...any) (count int64, err error)
	GetPrescriptionsByPatientID(query string, args ...any) ([]dto.PrescriptionListItem, error)
	GetPrescriptionByPatientIDCount(patientID string) (count int64, err error)
	GetPrescriptionsByDoctorID(doctorID string) ([]Prescription, error)
	//UpdatePrescription(prescription Prescription) error
	DeletePrescription(id string) error
	FindMany(query string, args ...any) ([]dto.PrescriptionListItem, error)
	FindByStatus(organisationID string, status string, limit int, offset int) ([]dto.PrescriptionListItem, error)
	CountByStatus(organisationID string, status string) (int64, error)
	FindPrescriptionByID(query string, id string) (presc Prescription, err error)
	UpdateStatus(tx *gorm.DB, status string, prescriptionID string) (err error)
	GetNotificationDetails(query string, prescriptionID string) (PrescriptionNotificationData, error)

	Count(query string, args ...any) (int64, error)
}

func (pdb *PrescriptionDB) CreatePrescription(db *gorm.DB, prescription Prescription) error {
	return db.Create(&prescription).Error
}

func (pdb *PrescriptionDB) GetPrescriptionByID(id string) (*Prescription, error) {
	var prescription Prescription
	err := pdb.db.First(&prescription, "id = ?", id).Error
	return &prescription, err
}

func (pdb *PrescriptionDB) GetPrescriptionsByAppointmentID(query string, cond ...any) ([]PrescriptionAppointmentData, error) {
	var prescriptions []PrescriptionAppointmentData

	err := pdb.db.Raw(query, cond...).Find(&prescriptions).Error
	if err != nil {
		return nil, err
	}
	return prescriptions, nil
}

func (pdb *PrescriptionDB) GetPrescriptionsByPatientID(query string, args ...any) ([]dto.PrescriptionListItem, error) {
	var prescriptions []dto.PrescriptionListItem
	if err := pdb.db.Raw(query, args...).Find(&prescriptions).Error; err != nil {
		return nil, err
	}
	return prescriptions, nil
}

func (pdb *PrescriptionDB) GetPrescriptionByPatientIDCount(patientID string) (count int64, err error) {
	err = pdb.db.Model(&Prescription{}).
		Where("patient_id = ?", patientID).
		Count(&count).
		Error
	return count, err
}

func (pdb *PrescriptionDB) GetPrescriptionsByDoctorID(doctorID string) ([]Prescription, error) {
	var prescriptions []Prescription
	err := pdb.db.Where("prescribed_by = ?", doctorID).Find(&prescriptions).Error
	return prescriptions, err
}

// func (pdb *PrescriptionDB) UpdatePrescription(prescription Prescription) error {
// 	return pdb.db.Exec("update prescriptions set medicines = ? , updated_at = ? where id = ?", prescription.Medicines, prescription.UpdatedAt, prescription.ID).Error
// }

func (pdb *PrescriptionDB) DeletePrescription(id string) error {
	return pdb.db.Delete(&Prescription{}, "id = ?", id).Error
}
func (pdb *PrescriptionDB) FindPrescriptionByID(query string, id string) (presc Prescription, err error) {
	err = pdb.db.Raw(query, id).Scan(&presc).Error
	if err != nil {
		return
	}
	return
}

func (pdb *PrescriptionDB) FindMany(query string, args ...any) (prescription []dto.PrescriptionListItem, err error) {
	err = pdb.db.Raw(query, args...).Scan(&prescription).Error
	if err != nil {
		return
	}
	return
}
func (pdb *PrescriptionDB) FindByStatus(organisationID string, status string, limit int, offset int) ([]dto.PrescriptionListItem, error) {
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
		return nil, err
	}
	return prescriptions, nil
}
func (pdb *PrescriptionDB) CountByStatus(organisationID string, status string) (int64, error) {
	var count int64
	err := pdb.db.Model(&Prescription{}).
		Where("organisation_id = ? AND status = ?", organisationID, status).
		Count(&count).Error
	return count, err
}
func (pdb *PrescriptionDB) UpdateStatus(db *gorm.DB, status string, prescriptionID string) (err error) {
	query := `UPDATE prescriptions
	SET status = ?, updated_at = ?
	WHERE id = ?;`
	err = db.Exec(query, status, time.Now(), prescriptionID).Error
	if err != nil {
		return
	}
	return
}
func (pdb *PrescriptionDB) Count(query string, args ...any) (int64, error) {
	var count int64
	err := pdb.db.Raw(query, args...).Scan(&count).Error
	return count, err
}
func (pdb *PrescriptionDB) GetPrescriptionByAppointmentIDCount(cond ...any) (count int64, err error) {
	err = pdb.db.
		Model(&Prescription{}).
		Where("appointment_id = ? AND organisation_id = ?", cond...).
		Count(&count).
		Error
	if err != nil {
		return
	}
	return
}

func (pdb *PrescriptionDB) GetNotificationDetails(query string, prescriptionID string) (PrescriptionNotificationData, error) {
	var data PrescriptionNotificationData
	err := pdb.db.Raw(query, prescriptionID).Scan(&data).Error
	if err != nil {
		return PrescriptionNotificationData{}, err
	}
	return data, nil
}
