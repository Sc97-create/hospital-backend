package authentication

import "hospital-backend/internal/employee"

type UserRepository interface {
	GetUserID(username string) (user *employee.User, err error)
	UpdateLastLoginAttempt(values ...any) error
}

func (A *AuthRepo) GetUserID(username string) (user *employee.User, err error) {
	query := `select id,password_hash,last_login_attempt from users where email_id=$1`
	err = A.db.Raw(query, username).Scan(&user).Error
	if err != nil {
		return
	}
	return
}

func (A *AuthRepo) UpdateLastLoginAttempt(values ...any) error {
	query := `update users set last_login_attempt = ? where id=$1`
	return A.db.Exec(query, values...).Error
}
