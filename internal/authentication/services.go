package authentication

import (
	"errors"
	"hospital-backend/internal/authentication/dto"
	"hospital-backend/internal/jwt"
	jwtAuth "hospital-backend/internal/jwt"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	Repo       UserRepository
	JwtService jwtAuth.JwtService
}

func NewService(repo AuthRepo, jwtService jwt.JwtService) UserService {
	return UserService{Repo: &repo, JwtService: jwtService}
}

func (a *UserService) Login(L dto.LoginUser) (dto.LoginResponse, error) {
	user, err := a.Repo.GetUserID(L.Username)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	err = a.validateCredentials(L)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	verified, err := a.comparePwd(user.PasswordHash, L.Password)
	if err != nil || !verified {
		return dto.LoginResponse{}, errors.New("invalid credentials")
	}
	token, err := a.JwtService.AccessToken(user.ID, user.OrganisationID)
	if err != nil {
		return dto.LoginResponse{}, errors.New("failed to generate access token")
	}
	refreshID, err := a.JwtService.RefreshtokenRepo.FindIDByUserID(user.ID)
	if err != nil {
		return dto.LoginResponse{}, errors.New("failed to check refresh token")
	}
	var refreshtoken string
	if refreshID != "" {
		//update existing row
		refreshtoken, err = a.JwtService.UpdateRefreshToken(refreshID, user.ID, user.OrganisationID)
		if err != nil {
			return dto.LoginResponse{}, errors.New("failed to update refresh token")
		}
	} else {
		refreshtoken, err = a.JwtService.RefreshToken(user.OrganisationID, user.ID)
		if err != nil {
			return dto.LoginResponse{}, errors.New("failed to generate refresh token")
		}
	}

	err = a.Repo.UpdateLastLoginAttempt(user.ID, user.LastLoginAttempt+1)
	if err != nil {
		return dto.LoginResponse{}, errors.New("failed to update last login Attempt")
	}
	response := dto.LoginResponse{}
	response.UserID = user.ID
	response.Token = token
	response.RefreshToken = refreshtoken
	return response, nil
}
func (a *UserService) validateCredentials(L dto.LoginUser) error {
	if L.Password == "" || L.Username == "" {
		return errors.New("invalid credentials")
	}
	return nil
}
func (a *UserService) comparePwd(DbPwd string, userPwd string) (verified bool, err error) {
	err = bcrypt.CompareHashAndPassword([]byte(DbPwd), []byte(userPwd))
	if err != nil {
		err = errors.New("password verification failed")
		return
	}
	return true, nil
}
func (a *UserService) RefreshToken(refreshToken string) (dto.LoginResponse, error) {
	tokenresp, err := a.JwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	loginresp := a.toLoginResp(tokenresp)
	return loginresp, nil
}
func (a *UserService) toLoginResp(tokenresp jwt.TokenResp) dto.LoginResponse {
	return dto.LoginResponse{
		Token:        tokenresp.AccessToken,
		RefreshToken: tokenresp.RefreshToken,
	}
}
