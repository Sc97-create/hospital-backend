package employee

import "gorm.io/gorm"

type EmployeeRepository interface {
	Create(*gorm.DB, *User) error
	Update(string, map[string]interface{}) (err error)
	DeleteOne(string) (err error)
	ReadMany(limit int, skip int) ([]User, error)
	ReadOne(id string) (*User, error)
	ReadDoctors(name string) ([]User, error)
}

func (E *EmployeeRepo) Create(tx *gorm.DB, employee *User) (err error) {
	return tx.Create(employee).Error

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

func (E *EmployeeRepo) ReadDoctors(name string) (u []User, err error) {
	query := E.db.Model(&User{}).Select("users.*").Joins("JOIN roles on roles.id = users.role_id").Where("roles.name = ?", "Doctor")
	if name != "" {
		query = query.Where("users.first_name ILIKE ? OR users.last_name ILIKE ?", "%"+name+"%", "%"+name+"%")
	}
	err = query.Find(&u).Error
	return u, err
}
