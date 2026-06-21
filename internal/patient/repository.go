package patient

import "context"

type PatientRepository interface {
	Create(*Patient) error
	ReadMany(limit int, offset int, organisationID string) ([]Patient, error)
	ReadOne(patientID string) (Patient, error)
	Count(organisationID string) (int64, error)
	GetPatientByID(ctx context.Context, patientID string, organisationID string) (*Patient, error)
	UpdatePatient(ctx context.Context, query string, args []interface{}) error
}

func (p *PatientRepo) Create(record *Patient) error {
	err := p.db.Create(&record).Error
	if err != nil {
		return err
	}
	return nil
}
func (p *PatientRepo) ReadMany(limit int, offset int, organisationID string) (patients []Patient, err error) {
	query := `select id,uh_id,name,gender,age,weight,mobile_number,email_id,last_visit_date,blood_group,status from patients where organisation_id=? limit ? offset ?`
	err = p.db.Raw(query, organisationID, limit, offset).Scan(&patients).Error
	if err != nil {
		return
	}
	return
}
func (p *PatientRepo) ReadOne(id string) (patient Patient, err error) {
	err = p.db.First(&patient, "id=?", id).Error
	if err != nil {
		return
	}
	return
}
func (p *PatientRepo) Count(organisationID string) (int64, error) {
	var count int64
	err := p.db.Model(&Patient{}).Where("organisation_id=?", organisationID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (p *PatientRepo) GetPatientByID(ctx context.Context, patientID string, organisationID string) (*Patient, error) {
	var patient Patient
	err := p.db.WithContext(ctx).
		Where("id = ? AND organisation_id = ?", patientID, organisationID).
		First(&patient).
		Error
	if err != nil {
		return nil, err
	}
	return &patient, nil
}

func (p *PatientRepo) UpdatePatient(ctx context.Context, query string, args []interface{}) error {
	return p.db.WithContext(ctx).
		Exec(query, args...).
		Error
}
