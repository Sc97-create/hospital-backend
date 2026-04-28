package jwt

import (
	"time"

	"gorm.io/gorm"
)

type RefreshTokenModel struct {
	DB *gorm.DB
}

func NewRefreshTokenModel(db *gorm.DB) *RefreshTokenModel {
	return &RefreshTokenModel{DB: db}
}

type RefreshtokenRepo interface {
	Insert(refreshToken *RefreshToken) error
	FindByID(ID string) (*RefreshToken, error)
	Update(refreshToken string, expiresAt time.Time, id string) error
	CheckIfExist(ID string) (int64, error)
	FindIDByUserID(userID string) (string, error)
}

func (t *RefreshTokenModel) Insert(refreshToken *RefreshToken) (err error) {
	err = t.DB.Create(refreshToken).Error
	if err != nil {
		return
	}
	return
}
func (t *RefreshTokenModel) FindByID(ID string) (refreshToken *RefreshToken, err error) {
	query := `select token_hash,expires_at,user_id from refresh_tokens where id=$1`
	err = t.DB.Raw(query, ID).Scan(&refreshToken).Error
	if err != nil {
		return
	}
	return
}
func (t *RefreshTokenModel) Update(refreshtoken string, expiresAt time.Time, id string) (err error) {
	var rToken RefreshToken
	err = t.DB.Model(&rToken).Where("id=?", id).Updates(
		map[string]interface{}{
			"token_hash": refreshtoken,
			"expires_at": expiresAt,
		},
	).Error
	if err != nil {
		return
	}
	return
}
func (t *RefreshTokenModel) CheckIfExist(ID string) (count int64, err error) {
	query := `select count(*) from refresh_tokens where user_id=$1`
	err = t.DB.Raw(query, ID).Count(&count).Error
	if err != nil {
		return
	}
	return
}
func (t *RefreshTokenModel) FindIDByUserID(userID string) (id string, err error) {
	query := `select id from refresh_tokens where user_id=$1`
	err = t.DB.Raw(query, userID).Scan(&id).Error
	if err != nil {
		if err.Error() == "record not found" {
			return "", nil
		}
		return
	}
	return
}
