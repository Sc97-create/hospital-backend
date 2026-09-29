package employee

import (
	"errors"

	"go.uber.org/zap"
)

var ErrEmployeeNotFound = errors.New("employee not found")

type EmployeeRepository interface {
	Create(log *zap.Logger, employee *User) error
	Update(log *zap.Logger, id string, update map[string]interface{}) (err error)
	DeleteOne(log *zap.Logger, id string) (err error)
	ReadMany(log *zap.Logger, limit int, skip int, organisationID string, search string) ([]EmployeeListRow, error)
	ReadOne(log *zap.Logger, id string) (*EmployeeListRow, error)
	ReadDoctors(log *zap.Logger, query string, args ...any) ([]User, error)
	Count(log *zap.Logger, organisationID string, search string) (int64, error)
	CountByCodePrefix(log *zap.Logger, organisationID string, prefix string) (int64, error)
	CountByActiveStatus(log *zap.Logger, organisationID string) (EmployeeStatusCountRow, error)
	FindRoleIDByUserID(log *zap.Logger, userID string) (string, error)
	FindOrganisationIDByUserID(log *zap.Logger, userID string) (string, error)
}

func (E *EmployeeRepo) Create(log *zap.Logger, employee *User) (err error) {
	log = ensureLog(log)
	err = E.db.Create(employee).Error
	if err != nil {
		logDBError(log, "Create", err)
		return
	}
	return
}

func (E *EmployeeRepo) Update(log *zap.Logger, id string, update map[string]interface{}) error {
	log = ensureLog(log)
	err := E.db.Model(&User{}).Where("id=?", id).Updates(update).Error
	if err != nil {
		logDBError(log, "Update", err)
		return err
	}
	return nil
}

func (E *EmployeeRepo) DeleteOne(log *zap.Logger, id string) (err error) {
	log = ensureLog(log)
	err = E.db.Where("id=?", id).Delete(User{}).Error
	if err != nil {
		logDBError(log, "DeleteOne", err)
		return
	}
	return
}

func (E *EmployeeRepo) ReadMany(log *zap.Logger, limit int, offset int, organisationID string, search string) (rows []EmployeeListRow, err error) {
	log = ensureLog(log)
	query := `SELECT u.id, u.employee_code, u.username, u.first_name, u.last_name, u.email_id, u.phone_number,
		u.organisation_id, u.role_id, r.name AS role_name, u.department_id, d.name AS department_name, u.is_active
		FROM users u
		LEFT JOIN roles r ON r.id = u.role_id
		LEFT JOIN departments d ON d.id = u.department_id
		WHERE u.organisation_id=?`
	args := []any{organisationID}
	query, args = appendEmployeeSearch(query, args, search)
	query += ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	err = E.db.Raw(query, args...).Scan(&rows).Error
	if err != nil {
		logDBError(log, "ReadMany", err)
	}
	return
}

func (E *EmployeeRepo) Count(log *zap.Logger, organisationID string, search string) (int64, error) {
	log = ensureLog(log)
	query := `SELECT COUNT(*) FROM users u WHERE u.organisation_id=?`
	args := []any{organisationID}
	query, args = appendEmployeeSearch(query, args, search)
	var count int64
	err := E.db.Raw(query, args...).Scan(&count).Error
	if err != nil {
		logDBError(log, "Count", err)
		return 0, err
	}
	return count, nil
}

func appendEmployeeSearch(query string, args []any, search string) (string, []any) {
	if search == "" {
		return query, args
	}
	pattern := "%" + search + "%"
	query += ` AND (u.employee_code ILIKE ? OR u.first_name ILIKE ? OR u.last_name ILIKE ?)`
	args = append(args, pattern, pattern, pattern)
	return query, args
}

func (E *EmployeeRepo) CountByCodePrefix(log *zap.Logger, organisationID string, prefix string) (int64, error) {
	log = ensureLog(log)
	var count int64
	err := E.db.Model(&User{}).Where("organisation_id=? AND employee_code LIKE ?", organisationID, prefix+"%").Count(&count).Error
	if err != nil {
		logDBError(log, "CountByCodePrefix", err)
		return 0, err
	}
	return count, nil
}

func (E *EmployeeRepo) CountByActiveStatus(log *zap.Logger, organisationID string) (EmployeeStatusCountRow, error) {
	log = ensureLog(log)
	query := `
		SELECT
			COUNT(*) FILTER (WHERE u.is_active = true) AS active,
			COUNT(*) FILTER (WHERE u.is_active = false) AS inactive
		FROM users u
		WHERE u.organisation_id = ?
	`
	var row EmployeeStatusCountRow
	err := E.db.Raw(query, organisationID).Scan(&row).Error
	if err != nil {
		logDBError(log, "CountByActiveStatus", err)
		return EmployeeStatusCountRow{}, err
	}
	return row, nil
}

func (E *EmployeeRepo) ReadOne(log *zap.Logger, id string) (*EmployeeListRow, error) {
	log = ensureLog(log)
	query := `SELECT u.id, u.employee_code, u.username, u.first_name, u.last_name, u.email_id, u.phone_number,
		u.organisation_id, u.role_id, r.name AS role_name, u.department_id, d.name AS department_name, u.is_active
		FROM users u
		LEFT JOIN roles r ON r.id = u.role_id
		LEFT JOIN departments d ON d.id = u.department_id
		WHERE u.id = ?`
	var row EmployeeListRow
	err := E.db.Raw(query, id).Scan(&row).Error
	if err != nil {
		logDBError(log, "ReadOne", err)
		return nil, err
	}
	if row.ID == "" {
		return nil, ErrEmployeeNotFound
	}
	return &row, nil
}

func (E *EmployeeRepo) ReadDoctors(log *zap.Logger, query string, args ...any) ([]User, error) {
	log = ensureLog(log)
	var users []User
	err := E.db.Raw(query, args...).Scan(&users).Error
	if err != nil {
		logDBError(log, "ReadDoctors", err)
	}
	return users, err
}

func (E *EmployeeRepo) FindOrganisationIDByUserID(log *zap.Logger, userID string) (string, error) {
	log = ensureLog(log)
	var organisationID string
	err := E.db.Raw(`SELECT organisation_id FROM users WHERE id = ?`, userID).Scan(&organisationID).Error
	if err != nil {
		logDBError(log, "FindOrganisationIDByUserID", err)
		return "", err
	}
	if organisationID == "" {
		return "", ErrEmployeeNotFound
	}
	return organisationID, nil
}

func (E *EmployeeRepo) FindRoleIDByUserID(log *zap.Logger, userID string) (string, error) {
	log = ensureLog(log)
	var roleID string
	err := E.db.Raw(`SELECT role_id FROM users WHERE id = ?`, userID).Scan(&roleID).Error
	if err != nil {
		logDBError(log, "FindRoleIDByUserID", err)
		return "", err
	}
	if roleID == "" {
		return "", ErrEmployeeNotFound
	}
	return roleID, nil
}
