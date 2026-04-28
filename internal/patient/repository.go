package patient

type PatientRepository interface {
	Create(*Patient) error
	ReadMany(limit int, offset int, organisationID string) ([]Patient, error)
	ReadOne(patientID string) (Patient, error)
	Count(organisationID string) (int64, error)
}

func (p *PatientRepo) Create(record *Patient) error {
	err := p.db.Create(&record).Error
	if err != nil {
		return err
	}
	return nil
}
func (p *PatientRepo) ReadMany(limit int, offset int, organisationID string) (patients []Patient, err error) {
	query := `select id,patient_code,first_name,last_name,gender,age,weight,mobile_number,email_id,admission_date,discharge_date,status from patients where organisation_id=$1 limit $2 offset $3`
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
