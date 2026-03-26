package utils

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	auth "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func CreateToken(userID string) (string, error) {
	todaytime := time.Now()
	expirytime := todaytime.Add(15 * time.Minute).Unix()
	claims := auth.MapClaims{
		"iss": "https://www.medipay.in",
		"sub": userID,
		"exp": expirytime,
		"aud": "mediPay",
		"iat": todaytime.Unix(),
		"jti": uuid.NewString(),
	}
	token := auth.NewWithClaims(auth.SigningMethodES256, claims)
	data, err := os.ReadFile(`C:\Users\sachin\Hospital-backend\config\keys\jwt_private.pem`)
	if err != nil {
		return "", err
	}
	privatekey, err := auth.ParseECPrivateKeyFromPEM(data)
	if err != nil {
		return "", err
	}
	signedToken, err := token.SignedString(privatekey)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}
func SetRefreshToken(userID string) (claims auth.Claims, err error) {
	//refreshT := new(RefreshToken)
	currentTime := time.Now()
	expirytime := currentTime.AddDate(0, 0, 20)
	claims = auth.MapClaims{
		"iss": "https://medipay.com",
		"exp": expirytime.Unix(),
		"iat": currentTime.Unix(),
		"aud": "medipay",
		"sub": userID,
	}
	return

}
func VerifyToken(refreshtoken string) (flag bool, claims auth.Claims, err error) {
	publicKey, err := os.ReadFile(`C:\Users\sachin\Hospital-backend\config\keys\jwt_public.pem`)
	if err != nil {
		return
	}
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return false, nil, errors.New("invalid public key PEM")
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, nil, err
	}

	pubKey, ok := pubInterface.(*ecdsa.PublicKey)
	if !ok {
		return false, nil, errors.New("not an ECDSA public key")
	}
	token, err := jwt.Parse(refreshtoken, func(t *auth.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, errors.New("unsupported signing method")
		}
		return pubKey, err
	})
	if err != nil {
		err = errors.New("fail to verify token")
		return
	}
	if !token.Valid {
		return false, nil, errors.New("invalid refresh token")
	}
	claims, ok = token.Claims.(jwt.MapClaims)
	if !ok {
		return false, nil, errors.New("invalid token claims")
	}
	return true, claims, nil
}
