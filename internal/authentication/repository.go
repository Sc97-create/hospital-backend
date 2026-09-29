package authentication

import (
	"errors"
	"hospital-backend/internal/employee"
	"time"

	"go.uber.org/zap"
)

var errUserNotFound = errors.New("user not found")

type UserRepository interface {
	GetUserID(log *zap.Logger, username string) (user *employee.User, err error)
	GetUserByID(log *zap.Logger, userID string) (user *employee.User, err error)
	GetUserByPasswordResetTokenHash(log *zap.Logger, tokenHash string) (user *employee.User, err error)
	UpdateLastLoginAttempt(log *zap.Logger, userID string, lastLoginAttempt int) error
	ClearTempPassword(log *zap.Logger, userID string) error
	UpdatePassword(log *zap.Logger, userID string, passwordHash string) error
	SavePasswordResetToken(log *zap.Logger, userID string, tokenHash string, lastPwdUpdated time.Time) error
}

func (A *AuthRepo) GetUserID(log *zap.Logger, username string) (*employee.User, error) {
	log = ensureLog(log)
	var user employee.User
	query := `
		SELECT
			id,
			password_hash,
			last_login_attempt,
			organisation_id,
			temp_password,
			role_id,
			email_id,
			first_name,
			last_name,
			password_reset_token_hash,
			last_pwd_updated
		FROM users
		WHERE email_id = $1
	`
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

func (A *AuthRepo) GetUserByID(log *zap.Logger, userID string) (*employee.User, error) {
	log = ensureLog(log)
	var user employee.User
	query := `
		SELECT
			id,
			password_hash,
			temp_password
		FROM users
		WHERE id = $1
	`
	err := A.db.Raw(query, userID).Scan(&user).Error
	if err != nil {
		log.Error("auth repo error", zap.String("op", "GetUserByID"), zap.Error(err))
		return nil, err
	}
	if user.ID == "" {
		return nil, errUserNotFound
	}
	return &user, nil
}

func (A *AuthRepo) GetUserByPasswordResetTokenHash(log *zap.Logger, tokenHash string) (*employee.User, error) {
	log = ensureLog(log)
	var user employee.User
	query := `
		SELECT id
		FROM users
		WHERE password_reset_token_hash = $1
	`
	err := A.db.Raw(query, tokenHash).Scan(&user).Error
	if err != nil {
		log.Error("auth repo error", zap.String("op", "GetUserByPasswordResetTokenHash"), zap.Error(err))
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
	query := `
		UPDATE users
		SET password_hash = $1,
		    temp_password = '',
		    password_reset_token_hash = '',
		    last_pwd_updated = NOW()
		WHERE id = $2
	`
	err := A.db.Exec(query, passwordHash, userID).Error
	if err != nil {
		log.Error("auth repo error", zap.String("op", "UpdatePassword"), zap.Error(err))
		return err
	}
	return nil
}

func (A *AuthRepo) SavePasswordResetToken(log *zap.Logger, userID string, tokenHash string, lastPwdUpdated time.Time) error {
	log = ensureLog(log)
	query := `
		UPDATE users
		SET password_reset_token_hash = $1,
		    last_pwd_updated = $2
		WHERE id = $3
	`
	err := A.db.Exec(query, tokenHash, lastPwdUpdated, userID).Error
	if err != nil {
		log.Error("auth repo error", zap.String("op", "SavePasswordResetToken"), zap.Error(err))
		return err
	}
	return nil
}
