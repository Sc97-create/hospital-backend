package authentication

import (
	"errors"
	"hospital-backend/internal/employee"

	"go.uber.org/zap"
)

var errUserNotFound = errors.New("user not found")

type UserRepository interface {
	GetUserID(log *zap.Logger, username string) (user *employee.User, err error)
	UpdateLastLoginAttempt(log *zap.Logger, userID string, lastLoginAttempt int) error
	ClearTempPassword(log *zap.Logger, userID string) error
	UpdatePassword(log *zap.Logger, userID string, passwordHash string) error
}

func (A *AuthRepo) GetUserID(log *zap.Logger, username string) (*employee.User, error) {
	log = ensureLog(log)
	var user employee.User
	query := `select id,password_hash,last_login_attempt,organisation_id,temp_password,role_id from users where email_id=$1`
	err := A.db.Raw(query, username).Scan(&user).Error
	if err != nil {
		log.Error("auth repo error", zap.String("op", "GetUserID"), zap.Error(err))
		return nil, err
	}
	if user.ID == "" {
		return nil, errUserNotFound
	}
	return &user, nil
}

func (A *AuthRepo) UpdateLastLoginAttempt(log *zap.Logger, userID string, lastLoginAttempt int) error {
	log = ensureLog(log)
	query := `update users set last_login_attempt = $1 where id=$2`
	err := A.db.Exec(query, lastLoginAttempt, userID).Error
	if err != nil {
		log.Error("auth repo error", zap.String("op", "UpdateLastLoginAttempt"), zap.Error(err))
		return err
	}
	return nil
}
func (A *AuthRepo) ClearTempPassword(log *zap.Logger, userID string) error {
	log = ensureLog(log)
	query := `UPDATE users SET temp_password = '' WHERE id = $1`
	err := A.db.Exec(query, userID).Error
	if err != nil {
		log.Error("auth repo error", zap.String("op", "ClearTempPassword"), zap.Error(err))
		return err
	}
	return nil
}

func (A *AuthRepo) UpdatePassword(log *zap.Logger, userID string, passwordHash string) error {
	log = ensureLog(log)
	query := `UPDATE users SET password_hash = $1, temp_password = '' WHERE id = $2`
	err := A.db.Exec(query, passwordHash, userID).Error
	if err != nil {
		log.Error("auth repo error", zap.String("op", "UpdatePassword"), zap.Error(err))
		return err
	}
	return nil
}
