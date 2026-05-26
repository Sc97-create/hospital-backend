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
	CreatePrescription(prescription Prescription) error
	GetPrescriptionByID(id string) (*Prescription, error)
	GetPrescriptionsByPatientID(patientID string) ([]Prescription, error)
	GetPrescriptionsByDoctorID(doctorID string) ([]Prescription, error)
	UpdatePrescription(prescription Prescription) error
	DeletePrescription(id string) error
	FindMany(limit int, offset int, organisationID string) ([]dto.PrescriptionListItem, error)
	FindPrescriptionByID(query string, id string) (presc Prescription, err error)
	UpdateStatus(status Status, prescriptionID string) (err error)
	Count(organisationID string) (int64, error)
}

func (pdb *PrescriptionDB) CreatePrescription(prescription Prescription) error {
	return pdb.db.Create(&prescription).Error
}

func (pdb *PrescriptionDB) GetPrescriptionByID(id string) (*Prescription, error) {
	var prescription Prescription
	err := pdb.db.First(&prescription, "id = ?", id).Error
	return &prescription, err
}

func (pdb *PrescriptionDB) GetPrescriptionsByPatientID(patientID string) ([]Prescription, error) {
	var prescriptions []Prescription
	err := pdb.db.Where("patient_id = ?", patientID).Find(&prescriptions).Error
	return prescriptions, err
}

func (pdb *PrescriptionDB) GetPrescriptionsByDoctorID(doctorID string) ([]Prescription, error) {
	var prescriptions []Prescription
	err := pdb.db.Where("prescribed_by = ?", doctorID).Find(&prescriptions).Error
	return prescriptions, err
}

func (pdb *PrescriptionDB) UpdatePrescription(prescription Prescription) error {
	return pdb.db.Exec("update prescriptions set medicines = ? , updated_at = ? where id = ?", prescription.Medicines, prescription.UpdatedAt, prescription.ID).Error
}

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

func (pdb *PrescriptionDB) FindMany(limit int, offset int, organisationID string) (prescription []dto.PrescriptionListItem, err error) {
	query := `SELECT p.id, p.code, e.username AS prescribed_by, p.created_at, p.status as status
	FROM prescriptions AS p
	JOIN users AS e ON p.prescribed_by = e.id
	WHERE p.organisation_id = ? LIMIT ? OFFSET ?`
	err = pdb.db.Raw(query, organisationID, limit, offset).Scan(&prescription).Error
	if err != nil {
		return
	}
	return
}
func (pdb *PrescriptionDB) UpdateStatus(status Status, prescriptionID string) (err error) {
	query := `UPDATE prescriptions
	SET status = ?, updated_at = ?
	WHERE id = ?;`
	err = pdb.db.Exec(query, status, time.Now(), prescriptionID).Error
	if err != nil {
		return
	}
	return
}
func (pdb *PrescriptionDB) Count(organisationID string) (int64, error) {
	var count int64
	err := pdb.db.Model(&Prescription{}).Where("organisation_id = ?", organisationID).Count(&count).Error
	return count, err
}
