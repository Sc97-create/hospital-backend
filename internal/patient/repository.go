package patient

type PatientRepository interface {
	Create(*Patient) error
	ReadMany(limit int, offset int) ([]Patient, error)
	ReadOne(patientID string) (Patient, error)
}

func (p *PatientRepo) Create(record *Patient) error {
	err := p.db.Create(&record).Error
	if err != nil {
		return err
	}
	return nil
}
func (p *PatientRepo) ReadMany(limit int, offset int) (patients []Patient, err error) {
	err = p.db.Find(&patients, "limit ? offset ?", limit, offset).Error
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
