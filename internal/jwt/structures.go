package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ExpiresAt int
type KeyPath string

const (
	AccessTokenExpiresAt  ExpiresAt = 15
	RefreshTokenExpiresAt ExpiresAt = 7
)

type RefreshToken struct {
	ID        string    `json:"id" gorm:"primaryKey; type:uuid"`
	UserID    string    `json:"user_id" gorm:"not null"`
	TokenHash string    `json:"token_hash" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
}
type claims struct {
	UserID string `json:"sub"`
	Claims jwt.RegisteredClaims
}
type TokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
type Claims struct {
	RefereshToken string    `json:"refresh_token"`
	ExpiresAt     time.Time `json:"expires_at"`
	JTI           string    `json:"jti"`
}
