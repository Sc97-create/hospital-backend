package jwt

type TokenRepo interface {
	FindByID(userID string) (*RefreshToken, error)
	Insert(rModel RefreshToken) error
	UpdateToken(*RefreshToken) error
	CheckIfExist(userID string) (int, error)
}

func (rt *Refreshtoken) Insert(rModel RefreshToken) (err error) {
	err = rt.DB.Create(rModel).Error
	if err != nil {
		return
	}
	return
}
func (rt *Refreshtoken) FindByID(userID string) (*RefreshToken, error) {
	var refreshToken RefreshToken
	query := `select refresh_token from refresh_tokens where user_id=$1`
	err := rt.DB.Raw(query, userID).Scan(&refreshToken).Error
	if err != nil {
		return nil, err
	}
	return &refreshToken, nil
}
func (rt *Refreshtoken) UpdateToken(refreshToken *RefreshToken) error {
	var rToken RefreshToken
	err := rt.DB.Model(&rToken).Where("user_id=?", refreshToken.UserID).Updates(
		map[string]interface{}{
			"refresh_token": refreshToken.RefreshToken,
		},
	).Error
	if err != nil {
		return err
	}
	return nil
}
func (rt *Refreshtoken) CheckIfExist(userID string) (int64, error) {
	var refreshModel RefreshToken
	query := `select count(*) from refresh_tokens`
	count := int64(0)
	err := rt.DB.Model(&refreshModel).Where(query, userID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
