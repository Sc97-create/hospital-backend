package employee

import (
	"errors"
)

var errEmployeeNotFound = errors.New("employee not found")

type EmployeeRepository interface {
	Create(*User) error
	Update(string, map[string]interface{}) (err error)
	DeleteOne(string) (err error)
	ReadMany(limit int, skip int, organisationID string, search string) ([]EmployeeListRow, error)
	ReadOne(id string) (*EmployeeListRow, error)
	ReadDoctors(query string, args ...any) ([]User, error)
	Count(organisationID string, search string) (int64, error)
	CountByCodePrefix(organisationID string, prefix string) (int64, error)
	FindRoleIDByUserID(userID string) (string, error)
}

func (E *EmployeeRepo) Create(employee *User) (err error) {
	err = E.db.Create(employee).Error
	if err != nil {
		return
	}
	return

}

func (E *EmployeeRepo) Update(id string, update map[string]interface{}) error {
	err := E.db.Model(&User{}).Where("id=?", id).Updates(update).Error
	if err != nil {
		return err
	}
	return nil
}
func (E *EmployeeRepo) DeleteOne(id string) (err error) {
	err = E.db.Where("id=?", id).Delete(User{}).Error
	if err != nil {
		return
	}
	return
}
func (E *EmployeeRepo) ReadMany(limit int, offset int, organisationID string, search string) (rows []EmployeeListRow, err error) {
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
	return
}

func (E *EmployeeRepo) Count(organisationID string, search string) (int64, error) {
	query := `SELECT COUNT(*) FROM users u WHERE u.organisation_id=?`
	args := []any{organisationID}
	query, args = appendEmployeeSearch(query, args, search)
	var count int64
	err := E.db.Raw(query, args...).Scan(&count).Error
	if err != nil {
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

func (E *EmployeeRepo) CountByCodePrefix(organisationID string, prefix string) (int64, error) {
	var count int64
	err := E.db.Model(&User{}).Where("organisation_id=? AND employee_code LIKE ?", organisationID, prefix+"%").Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
func (E *EmployeeRepo) ReadOne(id string) (*EmployeeListRow, error) {
	query := `SELECT u.id, u.employee_code, u.username, u.first_name, u.last_name, u.email_id, u.phone_number,
		u.organisation_id, u.role_id, r.name AS role_name, u.department_id, d.name AS department_name, u.is_active
		FROM users u
		LEFT JOIN roles r ON r.id = u.role_id
		LEFT JOIN departments d ON d.id = u.department_id
		WHERE u.id = ?`
	var row EmployeeListRow
	err := E.db.Raw(query, id).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, errEmployeeNotFound
	}
	return &row, nil
}

func (E *EmployeeRepo) ReadDoctors(query string, args ...any) ([]User, error) {
	var users []User
	err := E.db.Raw(query, args...).Scan(&users).Error
	return users, err
}

func (E *EmployeeRepo) FindRoleIDByUserID(userID string) (string, error) {
	var roleID string
	err := E.db.Raw(`SELECT role_id FROM users WHERE id = ?`, userID).Scan(&roleID).Error
	if err != nil {
		return "", err
	}
	if roleID == "" {
		return "", errEmployeeNotFound
	}
	return roleID, nil
}
