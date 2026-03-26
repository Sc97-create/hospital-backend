package jwt

import (
	"hospital-backend/shared/jwt/utils"
	"time"

	"github.com/google/uuid"
)

type TokenService struct {
	RefreshTokenRepo TokenRepo
}

func NewTokenService(TRepo TokenRepo) *TokenService {
	return &TokenService{RefreshTokenRepo: TRepo}
}
func (Tservice *TokenService) CreateRefreshToken(userID string) {
	// claims, err := utils.SetRefreshToken(userID)
	// if err != nil {
	// 	return
	// }
	token, err := utils.CreateToken(userID)
	if err != nil {
		return
	}
	var refreshModel RefreshToken
	refreshModel.ID = uuid.NewString()
	refreshModel.CreatedAt = time.Now()
	refreshModel.RefreshToken = token
	refreshModel.UserID = userID
	refreshModel.UpdatedAt = time.Now()
	err = Tservice.RefreshTokenRepo.Insert(refreshModel)
	if err != nil {
		return
	}

}
func (Tservice *TokenService) CheckIfRefreshToken(userID string) (bool, error) {
	count, err := Tservice.RefreshTokenRepo.CheckIfExist(userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func (Tservice *TokenService) UpdateRefreshToken(refreshToken string, userID string) error {
	var refreshModel RefreshToken
	refreshModel.UserID = userID
	refreshModel.RefreshToken = refreshToken
	err := Tservice.RefreshTokenRepo.UpdateToken(&refreshModel)
	if err != nil {
		return err
	}
	return nil
}
func (Tservice *TokenService) FindByID(userID string) (*RefreshToken, error) {
	refreshModel, err := Tservice.RefreshTokenRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return refreshModel, nil
}
func (Tservice *TokenService) VerifyRefreshToken(userID string) error {
	rTModel, err := Tservice.RefreshTokenRepo.FindByID(userID)
	if err != nil {
		return err
	}
	flag, _, err := utils.VerifyToken(rTModel.RefreshToken)
	if err != nil {
		return err
	}
	if flag {
		// need to find refresh token
	}
	return nil
}
