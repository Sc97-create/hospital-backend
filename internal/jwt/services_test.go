package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hospital-backend/config"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type mockRefreshTokenRepo struct {
	insertCalled   bool
	insertErr      error
	findByIDResult *RefreshToken
	findByIDErr    error
	updateCalled   bool
	updateErr      error
	lastInserted   *RefreshToken
	lastUpdateHash string
	lastUpdateID   string
}

func (m *mockRefreshTokenRepo) Insert(rt *RefreshToken) error {
	m.insertCalled = true
	m.lastInserted = rt
	return m.insertErr
}

func (m *mockRefreshTokenRepo) FindByID(ID string) (*RefreshToken, error) {
	return m.findByIDResult, m.findByIDErr
}

func (m *mockRefreshTokenRepo) Update(refreshToken string, expiresAt time.Time, id string) error {
	m.updateCalled = true
	m.lastUpdateHash = refreshToken
	m.lastUpdateID = id
	_ = expiresAt
	return m.updateErr
}

func (m *mockRefreshTokenRepo) CheckIfExist(ID string) (int64, error) {
	return 0, nil
}

func (m *mockRefreshTokenRepo) FindIDByUserID(userID string) (string, error) {
	return "", nil
}

func (m *mockRefreshTokenRepo) DeleteByID(id string) error {
	return nil
}

func (m *mockRefreshTokenRepo) DeleteByUserID(userID string) error {
	return nil
}

func setupTestKeys(t *testing.T) (*config.Config, func()) {
	t.Helper()

	dir := t.TempDir()
	privRel := filepath.Join("testdata-keys", "private.pem")
	pubRel := filepath.Join("testdata-keys", "public.pem")
	privPath := filepath.Join(dir, privRel)
	pubPath := filepath.Join(dir, pubRel)

	if err := os.MkdirAll(filepath.Dir(privPath), 0o755); err != nil {
		t.Fatalf("mkdir keys: %v", err)
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	privBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	if err := os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg := &config.Config{
		PrivateKeyPath: privRel,
		PublicKeyPath:  pubRel,
	}
	return cfg, func() {
		_ = os.Chdir(origWD)
	}
}

func newService(repo RefreshtokenRepo, cfg *config.Config) *JwtService {
	return NewJwtService(repo, cfg)
}

func TestAccessToken_Success(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{}, cfg)
	token, err := svc.AccessToken("user-123", "org-456")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty access token")
	}

	parsed, err := svc.parseToken(token)
	if err != nil {
		t.Fatalf("failed to parse generated access token: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}
	if claims["sub"] != "user-123" {
		t.Errorf("expected sub=user-123, got %v", claims["sub"])
	}
	if claims["iss"] != "org-456" {
		t.Errorf("expected iss=org-456, got %v", claims["iss"])
	}
}

func TestAccessToken_ContainsExpiry(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{}, cfg)
	token, err := svc.AccessToken("user-1", "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, _ := svc.parseToken(token)
	claims := parsed.Claims.(jwt.MapClaims)
	exp, err := claims.GetExpirationTime()
	if err != nil {
		t.Fatalf("expected expiration claim: %v", err)
	}

	expectedExp := time.Now().Add(time.Duration(AccessTokenExpiresAt) * time.Minute)
	diff := expectedExp.Sub(exp.Time)
	if diff < -30*time.Second || diff > 30*time.Second {
		t.Errorf("expiry not within expected range: got %v, expected ~%v", exp.Time, expectedExp)
	}
}

func TestAccessToken_MissingPrivateKey(t *testing.T) {
	cfg := &config.Config{
		PrivateKeyPath: "missing/private.pem",
		PublicKeyPath:  "missing/public.pem",
	}
	svc := newService(&mockRefreshTokenRepo{}, cfg)
	_, err := svc.AccessToken("u", "o")
	if err == nil {
		t.Fatal("expected error when private key file is missing")
	}
}

func TestRefreshToken_Success(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{}, cfg)
	claims, err := svc.RefreshToken("org-1", "user-1", "refresh-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.RefereshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if claims.JTI != "refresh-1" {
		t.Errorf("expected jti=refresh-1, got %q", claims.JTI)
	}
}

func TestRefreshToken_TokenIsParseable(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{}, cfg)
	claimsModel, err := svc.RefreshToken("org-1", "user-1", "refresh-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := svc.parseToken(claimsModel.RefereshToken)
	if err != nil {
		t.Fatalf("refresh token should be parseable: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != "user-1" {
		t.Errorf("expected sub=user-1, got %v", claims["sub"])
	}
	if claims["iss"] != "org-1" {
		t.Errorf("expected iss=org-1, got %v", claims["iss"])
	}
}

func TestParseToken_TamperedToken(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{}, cfg)
	token, _ := svc.AccessToken("user-1", "org-1")
	tampered := token[:len(token)-5] + "XXXXX"
	if _, err := svc.parseToken(tampered); err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestCheckRefreshToken_Expired(t *testing.T) {
	svc := newService(&mockRefreshTokenRepo{}, &config.Config{})
	rt := &RefreshToken{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	err := svc.checkRefreshToken(rt, "user-1")
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
	if err.Error() != "token is expired" {
		t.Errorf("expected 'token is expired', got %q", err.Error())
	}
}

func TestCheckRefreshToken_UserMismatch(t *testing.T) {
	svc := newService(&mockRefreshTokenRepo{}, &config.Config{})
	rt := &RefreshToken{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := svc.checkRefreshToken(rt, "user-999"); err == nil {
		t.Fatal("expected error for user ID mismatch")
	}
}

func TestValidateRefreshToken_Success(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{}
	svc := newService(mock, cfg)

	claimsModel, err := svc.RefreshToken("org-1", "user-1", "refresh-1")
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}
	mock.findByIDResult = &RefreshToken{
		ID:        "refresh-1",
		UserID:    "user-1",
		TokenHash: claimsModel.RefereshToken,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	resp, err := svc.ValidateRefreshToken(zap.NewNop(), claimsModel.RefereshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if !mock.updateCalled {
		t.Error("expected Update to be called on the repo")
	}
}

func TestValidateRefreshToken_InvalidTokenString(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{}, cfg)
	if _, err := svc.ValidateRefreshToken(zap.NewNop(), "garbage-token"); err == nil {
		t.Fatal("expected error for invalid token string")
	}
}

func TestValidateRefreshToken_FindByIDError(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{findByIDErr: errors.New("db error")}
	svc := newService(mock, cfg)
	claimsModel, _ := svc.RefreshToken("org-1", "user-1", "refresh-1")
	if _, err := svc.ValidateRefreshToken(zap.NewNop(), claimsModel.RefereshToken); err == nil {
		t.Fatal("expected error when FindByID fails")
	}
}

func TestGetPrivateKey_Success(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{}, cfg)
	abs := filepath.Join(".", cfg.PrivateKeyPath)
	key, err := svc.getPrivateKey(KeyPath(abs))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil private key")
	}
}

func TestToken_UsesES256(t *testing.T) {
	cfg, cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{}, cfg)
	token, err := svc.AccessToken("u", "o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return svc.getPublickey()
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Method.Alg() != jwt.SigningMethodES256.Alg() {
		t.Errorf("expected ES256, got %s", parsed.Method.Alg())
	}
}
