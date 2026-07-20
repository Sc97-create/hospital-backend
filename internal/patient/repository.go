package patient

type PatientRepository interface {
	Create(*Patient) error
	ReadMany(limit int, offset int, organisationID string) ([]Patient, error)
	ReadOne(patientID string) (Patient, error)
	Count(organisationID string) (int64, error)
	ReadOneWithOrganisationID(query string, args ...any) (map[string]interface{}, error)
}

func (p *PatientRepo) Create(record *Patient) error {
	err := p.db.Create(&record).Error
	if err != nil {
		return err
	}
	return nil
}
func (p *PatientRepo) ReadMany(limit int, offset int, organisationID string) (patients []Patient, err error) {
	query := `select id,uh_id,name,gender,age,weight,mobile_number,email_id,last_visit_date,blood_group,status,created_at from patients where organisation_id=? limit ? offset ?`
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
func (p *PatientRepo) ReadOneWithOrganisationID(query string, args ...any) (patient map[string]interface{}, err error) {
	err = p.db.Raw(query, args...).Scan(&patient).Error
	if err != nil {
		return nil, err
	}
	return patient, nil
}
