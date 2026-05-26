package employee

type EmployeeRepository interface {
	Create(*User) error
	Update(string, map[string]interface{}) (err error)
	DeleteOne(string) (err error)
	ReadMany(limit int, skip int) ([]User, error)
	ReadOne(id string) (*User, error)
	ReadDoctors(query string, args ...any) ([]User, error)
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
func (E *EmployeeRepo) ReadMany(limit int, offset int) (u []User, err error) {
	err = E.db.Find(&u, "limit ?,offset ?", limit, offset).Error
	if err != nil {
		return
	}
	return
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
