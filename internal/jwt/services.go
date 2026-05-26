package jwt

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JwtService struct {
	RefreshtokenRepo RefreshtokenRepo
}

func NewJwtService(refreshRepo RefreshtokenRepo) *JwtService {
	return &JwtService{RefreshtokenRepo: refreshRepo}
}

func (j *JwtService) InsertRefreshToken(token string, expiry time.Time, userID string) error {
	var refreshToken RefreshToken
	refreshToken.TokenHash = token
	refreshToken.UserID = userID
	refreshToken.ExpiresAt = expiry
	refreshToken.ID = uuid.NewString()
	refreshToken.CreatedAt = time.Now()
	err := j.RefreshtokenRepo.Insert(&refreshToken)
	if err != nil {
		return err
	}
	return nil
}
func (j *JwtService) AccessToken(userID string, organistionID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    organistionID,
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(AccessTokenExpiresAt) * time.Minute)),
		Audience:  []string{"patient", "employee", "admin", "doctor"},
		ID:        uuid.NewString(),
	}
	return j.token(claims)
}
func (j *JwtService) RefreshToken(organisationID string, userID string, refreshID string) (Claims, error) {
	if refreshID == "" {
		refreshID = uuid.NewString()
	}
	claims := j.createclaims(organisationID, userID, refreshID)
	token, err := j.token(claims)
	if err != nil {
		return Claims{}, err
	}
	Claims := j.toClaimsModel(token, claims)
	return Claims, nil

}
func (j *JwtService) toClaimsModel(token string, claims jwt.RegisteredClaims) Claims {
	return Claims{
		RefereshToken: token,
		ExpiresAt:     claims.ExpiresAt.Time,
		JTI:           claims.ID,
	}
}
func (j *JwtService) UpdateRefreshToken(refreshID string, userID string, organisationID string) (rToken string, err error) {
	claims := j.createclaims(organisationID, userID, refreshID)
	token, err := j.token(claims)
	err = j.RefreshtokenRepo.Update(token, claims.ExpiresAt.Time, refreshID)
	if err != nil {
		return "", err
	}
	return token, nil
}
func (j *JwtService) ValidateRefreshToken(token string) (TokenResp, error) {
	var tokenResponse TokenResp
	token1, err := j.parseToken(token)
	if err != nil {
		return TokenResp{}, err
	}
	claims := token1.Claims.(jwt.MapClaims)
	refreshID, _ := claims["jti"].(string)
	userID, _ := claims["sub"].(string)
	organisationID, _ := claims["iss"].(string)
	refreshToken, err := j.RefreshtokenRepo.FindByID(refreshID)
	if err != nil {
		return TokenResp{}, err
	}
	err = j.checkRefreshToken(refreshToken, userID)
	if err != nil {
		return TokenResp{}, err
	}
	claimsModel, err := j.RefreshToken(organisationID, userID, refreshID)
	if err != nil {
		return TokenResp{}, err
	}
	newAccessToken, err := j.AccessToken(userID, organisationID)
	if err != nil {
		return TokenResp{}, err
	}
	refreshToken.ExpiresAt = time.Now().AddDate(0, 0, int(RefreshTokenExpiresAt))
	err = j.RefreshtokenRepo.Update(claimsModel.RefereshToken, refreshToken.ExpiresAt, refreshID)
	if err != nil {
		return TokenResp{}, err
	}
	tokenResponse.AccessToken = newAccessToken
	tokenResponse.RefreshToken = claimsModel.RefereshToken

	return tokenResponse, nil

}
func (j *JwtService) ValidateAccessToken(token string) (bool, error) {
	token1, err := j.parseToken(token)
	if err != nil {
		return false, err
	}
	claims := token1.Claims.(jwt.MapClaims)
	expiryTime := claims["exp"].(float64)
	if time.Unix(int64(expiryTime), 0).Before(time.Now()) {
		return false, errors.New("token is expired")
	}
	return true, nil

}
func (j *JwtService) CheckIfExist(userID string) (bool, error) {
	count, err := j.RefreshtokenRepo.CheckIfExist(userID)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, errors.New("refresh token not found")
	}
	return true, nil
}
func (j *JwtService) FindIDByUserID(userID string) (string, error) {
	id, err := j.RefreshtokenRepo.FindIDByUserID(userID)
	if err != nil {
		return "", err
	}
	return id, nil
}
func (j *JwtService) createclaims(organisationID string, userID string, refreshID string) jwt.RegisteredClaims {
	claims := jwt.RegisteredClaims{
		Issuer:    organisationID,
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(0, 0, int(RefreshTokenExpiresAt))),
		ID:        refreshID,
	}
	return claims
}

func (j *JwtService) toRefreshTokenModel(token string, id string, userID string, Expiry time.Time) (*RefreshToken, error) {
	refreshToken := RefreshToken{
		TokenHash: token,
		ID:        id,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: Expiry,
	}
	return &refreshToken, nil
}
func (j *JwtService) token(claims jwt.RegisteredClaims) (string, error) {
	privateKey, err := j.getPrivateKey(PrivateKeyPath)
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = uuid.NewString()
	accesstoken, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}
	return accesstoken, nil
}
func (j *JwtService) getPrivateKey(path KeyPath) (*ecdsa.PrivateKey, error) {
	key, err := os.ReadFile(string(path))
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key")
	}
	privateKey, err := x509.ParseECPrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EC private key: %w", err)
	}
	return privateKey, nil
}

func (j *JwtService) parseToken(token string) (*jwt.Token, error) {
	token1, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return j.getPublickey()
	})
	if err != nil {
		return nil, err
	}
	if !token1.Valid {
		return nil, errors.New("invalid token")
	}
	return token1, nil
}
func (j *JwtService) checkRefreshToken(refreshToken *RefreshToken, userID string) (err error) {
	if refreshToken == nil {
		err = errors.New("refresh token not found")
		return err
	}
	if refreshToken.ExpiresAt.Before(time.Now()) {
		err = errors.New("token is expired")
		return err
	}
	if refreshToken.UserID != userID {
		err = errors.New("")
		return err
	}
	return
}
func (j *JwtService) getPublickey() (*ecdsa.PublicKey, error) {
	key, err := os.ReadFile(string(PublicKeyPath))
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block from public key")
	}
	parsedKey, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	ecdsaKey, ok := parsedKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ECDSA")
	}
	return ecdsaKey, nil
}

//createrefreshtoken
//validaterefreshtoken
//validateacccesstoken
//getrefreshbyid
