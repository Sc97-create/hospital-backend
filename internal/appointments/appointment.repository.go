package appointments

import "gorm.io/gorm"

type AppointmentRepository interface {
	Create(appointment *Appointment) error
	GetAppointmentsByIDs(query string, cond ...any) ([]Appointment, error)
	FindManyByOrganisationID(query string, cond ...any) ([]map[string]interface{}, error)
	GetTotalAppointmentsByOrgID(query string, cond ...any) (int, error)
	GetAppointmentsPreview(query string, cond ...any) (map[string]interface{}, error)
	GetAppointmentCount(query string, cond ...any) (count int64, err error)
	GetAppointmentByID(appointmentID string) (Appointment, error)
	UpdateStatus(tx *gorm.DB, value interface{}, cond ...any) (err error)
	GetAppointmentByPatientID(query string, cond ...any) ([]map[string]interface{}, error)
	GetAppointmentByPatientIDCount(cond ...any) (count int64, err error)
	GetNotificationsDetails(query string, cond ...any) (map[string]interface{}, error)
}

func (r *CommonDB) Create(appointment *Appointment) error {
	return r.db.Create(appointment).Error
}

func (r *CommonDB) GetAppointmentsByIDs(query string, cond ...any) ([]Appointment, error) {
	var Appointments []Appointment
	err := r.db.Raw(query, cond...).Find(&Appointments).Error
	if err != nil {
		return nil, err
	}
	return Appointments, nil
}
func (r *CommonDB) FindManyByOrganisationID(query string, cond ...any) ([]map[string]interface{}, error) {
	var data []map[string]interface{}
	err := r.db.Raw(query, cond...).Scan(&data).Error
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (r *CommonDB) GetTotalAppointmentsByOrgID(query string, cond ...any) (int, error) {
	var count int
	err := r.db.Raw(query, cond...).Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
func (r *CommonDB) GetAppointmentsPreview(query string, cond ...any) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := r.db.Raw(query, cond...).Scan(&data).Error
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (r *CommonDB) GetAppointmentCount(query string, cond ...any) (count int64, err error) {
	err = r.db.Raw(query).Count(&count).Error
	if err != nil {
		return
	}
	return
}
func (r *CommonDB) GetAppointmentByID(appointmentID string) (Appointment, error) {
	var appointment Appointment
	err := r.db.Model(Appointment{}).Where("id=?", appointmentID).Scan(&appointment).Error
	if err != nil {
		return Appointment{}, err
	}
	return appointment, nil
}
func (r *CommonDB) UpdateStatus(tx *gorm.DB, value interface{}, cond ...any) (err error) {
	err = tx.Model(&Appointment{}).Where("id = ?", cond...).Update("status", value).Error
	if err != nil {
		return
	}
	return
}
func (r *CommonDB) GetAppointmentByPatientID(query string, cond ...any) (appointments []map[string]interface{}, err error) {
	err = r.db.Raw(query, cond...).Find(&appointments).Error
	if err != nil {
		return
	}
	return
}
func (r *CommonDB) GetAppointmentByPatientIDCount(cond ...any) (count int64, err error) {
	err = r.db.Model(&Appointment{}).Where("patient_id = ? and organisation_id = ?", cond...).Count(&count).Error
	if err != nil {
		return
	}
	return
}
func (r *CommonDB) GetNotificationsDetails(query string, cond ...any) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := r.db.Raw(query, cond...).First(&data).Error
	if err != nil {
		return nil, err
	}
	return data, nil
}
