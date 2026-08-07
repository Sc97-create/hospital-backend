package appointments

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AppointmentRepository interface {
	Create(log *zap.Logger, appointment *Appointment) error
	GetAppointmentsByIDs(log *zap.Logger, query string, cond ...any) ([]Appointment, error)
	FindManyByOrganisationID(log *zap.Logger, query string, cond ...any) ([]map[string]interface{}, error)
	GetTotalAppointmentsByOrgID(log *zap.Logger, query string, cond ...any) (int, error)
	GetAppointmentsPreview(log *zap.Logger, query string, cond ...any) (map[string]interface{}, error)
	GetAppointmentCount(log *zap.Logger, query string, cond ...any) (count int64, err error)
	GetAppointmentByID(log *zap.Logger, appointmentID string) (Appointment, error)
	UpdateStatus(log *zap.Logger, tx *gorm.DB, value interface{}, cond ...any) (err error)
	GetAppointmentByPatientID(log *zap.Logger, query string, cond ...any) ([]map[string]interface{}, error)
	GetAppointmentByPatientIDCount(log *zap.Logger, cond ...any) (count int64, err error)
	GetNotificationsDetails(log *zap.Logger, query string, cond ...any) (map[string]interface{}, error)
}

func (r *CommonDB) Create(log *zap.Logger, appointment *Appointment) error {
	log = ensureLog(log)
	err := r.db.Create(appointment).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "Create"), zap.Error(err))
		return err
	}
	return nil
}

func (r *CommonDB) GetAppointmentsByIDs(log *zap.Logger, query string, cond ...any) ([]Appointment, error) {
	log = ensureLog(log)
	var Appointments []Appointment
	err := r.db.Raw(query, cond...).Find(&Appointments).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "GetAppointmentsByIDs"), zap.Error(err))
		return nil, err
	}
	return Appointments, nil
}

func (r *CommonDB) FindManyByOrganisationID(log *zap.Logger, query string, cond ...any) ([]map[string]interface{}, error) {
	log = ensureLog(log)
	var data []map[string]interface{}
	err := r.db.Raw(query, cond...).Scan(&data).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "FindManyByOrganisationID"), zap.Error(err))
		return nil, err
	}
	return data, nil
}

func (r *CommonDB) GetTotalAppointmentsByOrgID(log *zap.Logger, query string, cond ...any) (int, error) {
	log = ensureLog(log)
	var count int
	err := r.db.Raw(query, cond...).Scan(&count).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "GetTotalAppointmentsByOrgID"), zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (r *CommonDB) GetAppointmentsPreview(log *zap.Logger, query string, cond ...any) (map[string]interface{}, error) {
	log = ensureLog(log)
	var data map[string]interface{}
	err := r.db.Raw(query, cond...).Scan(&data).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "GetAppointmentsPreview"), zap.Error(err))
		return nil, err
	}
	return data, nil
}

func (r *CommonDB) GetAppointmentCount(log *zap.Logger, query string, cond ...any) (count int64, err error) {
	log = ensureLog(log)
	err = r.db.Raw(query).Count(&count).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "GetAppointmentCount"), zap.Error(err))
		return
	}
	return
}

func (r *CommonDB) GetAppointmentByID(log *zap.Logger, appointmentID string) (Appointment, error) {
	log = ensureLog(log)
	var appointment Appointment
	err := r.db.Model(Appointment{}).Where("id=?", appointmentID).Scan(&appointment).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "GetAppointmentByID"), zap.Error(err))
		return Appointment{}, err
	}
	return appointment, nil
}

func (r *CommonDB) UpdateStatus(log *zap.Logger, tx *gorm.DB, value interface{}, cond ...any) (err error) {
	log = ensureLog(log)
	err = tx.Model(&Appointment{}).Where("id = ?", cond...).Update("status", value).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "UpdateStatus"), zap.Error(err))
		return
	}
	return
}

func (r *CommonDB) GetAppointmentByPatientID(log *zap.Logger, query string, cond ...any) (appointments []map[string]interface{}, err error) {
	log = ensureLog(log)
	err = r.db.Raw(query, cond...).Find(&appointments).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "GetAppointmentByPatientID"), zap.Error(err))
		return
	}
	return
}

func (r *CommonDB) GetAppointmentByPatientIDCount(log *zap.Logger, cond ...any) (count int64, err error) {
	log = ensureLog(log)
	err = r.db.Model(&Appointment{}).Where("patient_id = ? and organisation_id = ?", cond...).Count(&count).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "GetAppointmentByPatientIDCount"), zap.Error(err))
		return
	}
	return
}

func (r *CommonDB) GetNotificationsDetails(log *zap.Logger, query string, cond ...any) (map[string]interface{}, error) {
	log = ensureLog(log)
	var data map[string]interface{}
	err := r.db.Raw(query, cond...).First(&data).Error
	if err != nil {
		log.Error("appointment repo error", zap.String("op", "GetNotificationsDetails"), zap.Error(err))
		return nil, err
	}
	return data, nil
}
