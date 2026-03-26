package authentication

import (
	"errors"
	"hospital-backend/internal/authentication/dto"
	empUtils "hospital-backend/internal/employee/utils"
	"hospital-backend/shared/jwt/utils"
)

type UserService struct {
	Repo UserRepository
}

func NewService(repo AuthRepo) UserService {
	return UserService{Repo: &repo}
}

func (a *UserService) Login(L dto.LoginUser) (dto.LoginResponse, error) {
	user, err := a.Repo.GetUserID(L.Username)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	verified, err := empUtils.ComparePwd(user.PasswordHash, L.Password)
	if err != nil || !verified {
		return dto.LoginResponse{}, errors.New("invalid credentials")
	}
	err = a.Repo.UpdateLastLoginAttempt(user.ID, user.LastLoginAttempt+1)
	if err != nil {
		return dto.LoginResponse{}, errors.New("failed to update last login Attempt")
	}

	token, err := utils.CreateToken(user.ID)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	response := dto.LoginResponse{}
	response.UserID = user.ID
	response.Token = token
	return response, nil
}
