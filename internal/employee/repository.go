package employee

type EmployeeRepository interface {
	Create(*User) error
	Update(string, map[string]interface{}) (err error)
	DeleteOne(string) (err error)
	ReadMany(limit int, skip int, organisationID string) ([]User, error)
	ReadOne(id string) (*User, error)
	ReadDoctors(query string, args ...any) ([]User, error)
	Count(organisationID string) (int64, error)
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
func (E *EmployeeRepo) ReadMany(limit int, offset int, organisationID string) (u []User, err error) {
	query := `select id, username, first_name, last_name, email_id, phone_number, organisation_id, role_id, department_id, is_active from users where organisation_id=? limit ? offset ?`
	err = E.db.Raw(query, organisationID, limit, offset).Scan(&u).Error
	if err != nil {
		return
	}
	return
}

func (E *EmployeeRepo) Count(organisationID string) (int64, error) {
	var count int64
	err := E.db.Model(&User{}).Where("organisation_id=?", organisationID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
func (E *EmployeeRepo) ReadOne(id string) (*User, error) {
	var u User
	err := E.db.Find(&u, "id=?", id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (E *EmployeeRepo) ReadDoctors(query string, args ...any) ([]User, error) {
	var users []User
	err := E.db.Raw(query, args...).Scan(&users).Error
	return users, err
}
