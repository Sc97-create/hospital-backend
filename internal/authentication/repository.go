package authentication

import "hospital-backend/internal/employee"

type UserRepository interface {
	GetUserID(username string) (user *employee.User, err error)
	UpdateLastLoginAttempt(userID string, lastLoginAttempt int) error
}

func (A *AuthRepo) GetUserID(username string) (user *employee.User, err error) {
	query := `select id,password_hash,last_login_attempt, organisation_id from users where email_id=$1`
	err = A.db.Raw(query, username).Scan(&user).Error
	if err != nil {
		return
	}
	return
}

func (A *AuthRepo) UpdateLastLoginAttempt(userID string, lastLoginAttempt int) error {
	query := `update users set last_login_attempt = $1 where id=$2`
	return A.db.Exec(query, lastLoginAttempt, userID).Error
}
