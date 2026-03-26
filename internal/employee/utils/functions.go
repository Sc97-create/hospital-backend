package utils

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (hashedPwd []byte, err error) {
	hashedPwd, err = bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		err = errors.New("something went wrong, please contact administrator")
		return
	}
	return
}
func ComparePwd(DbPwd string, userPwd string) (verified bool, err error) {
	err = bcrypt.CompareHashAndPassword([]byte(DbPwd), []byte(userPwd))
	if err != nil {
		err = errors.New("password verification failed")
		return
	}
	return true, nil
}
