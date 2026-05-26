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

	"github.com/golang-jwt/jwt/v5"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockRefreshTokenRepo struct {
	insertCalled    bool
	insertErr       error
	findByIDResult  *RefreshToken
	findByIDErr     error
	updateCalled    bool
	updateErr       error
	checkIfExistCnt int64
	checkIfExistErr error

	// capture what was inserted/updated
	lastInserted *RefreshToken
	lastUpdated  *RefreshToken
}

func (m *mockRefreshTokenRepo) Insert(rt *RefreshToken) error {
	m.insertCalled = true
	m.lastInserted = rt
	return m.insertErr
}

func (m *mockRefreshTokenRepo) FindByID(ID string) (*RefreshToken, error) {
	return m.findByIDResult, m.findByIDErr
}

func (m *mockRefreshTokenRepo) Update(rt *RefreshToken) error {
	m.updateCalled = true
	m.lastUpdated = rt
	return m.updateErr
}

func (m *mockRefreshTokenRepo) CheckIfExist(ID string) (int64, error) {
	return m.checkIfExistCnt, m.checkIfExistErr
}

// ---------------------------------------------------------------------------
// Test helpers — generate EC P-256 key pair and write to the paths
// used by the constants PrivateKeyPath / PublicKeyPath.
// Backs up existing key files and restores them on cleanup.
// ---------------------------------------------------------------------------

func setupTestKeys(t *testing.T) (cleanup func()) {
	t.Helper()

	privPath := string(PrivateKeyPath)
	pubPath := string(PublicKeyPath)

	// Back up existing keys if they exist
	origPriv, privExists := backupFile(t, privPath)
	origPub, pubExists := backupFile(t, pubPath)

	// Generate EC P-256 key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate EC key: %v", err)
	}

	// Ensure directory exists
	keysDir := filepath.Dir(privPath)
	if err := os.MkdirAll(keysDir, 0o755); err != nil {
		t.Fatalf("failed to create keys dir: %v", err)
	}

	// Write private key (EC PRIVATE KEY)
	privBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal EC private key: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}

	// Write public key (PUBLIC KEY — PKIX format)
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal EC public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	if err := os.WriteFile(pubPath, pubPEM, 0o644); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	return func() {
		restoreFile(privPath, origPriv, privExists)
		restoreFile(pubPath, origPub, pubExists)
	}
}

func backupFile(t *testing.T, path string) ([]byte, bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func restoreFile(path string, data []byte, existed bool) {
	if existed {
		os.WriteFile(path, data, 0o600)
	}
}

func newService(repo RefreshtokenRepo) *JwtService {
	return NewJwtService(repo)
}

// ===========================================================================
// AccessToken tests
// ===========================================================================

func TestAccessToken_Success(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, err := svc.AccessToken("user-123", "org-456")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty access token")
	}

	// Parse and validate claims
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
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
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

	// Should expire roughly 15 minutes from now (allow 30s tolerance)
	expectedExp := time.Now().Add(time.Duration(AccessTokenExpiresAt) * time.Minute)
	diff := expectedExp.Sub(exp.Time)
	if diff < -30*time.Second || diff > 30*time.Second {
		t.Errorf("expiry not within expected range: got %v, expected ~%v", exp.Time, expectedExp)
	}
}

func TestAccessToken_ContainsAudience(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, _ := svc.AccessToken("u", "o")
	parsed, _ := svc.parseToken(token)
	claims := parsed.Claims.(jwt.MapClaims)

	aud, err := claims.GetAudience()
	if err != nil {
		t.Fatalf("expected audience claim: %v", err)
	}

	expectedAud := []string{"patient", "employee", "admin", "doctor"}
	if len(aud) != len(expectedAud) {
		t.Fatalf("expected %d audiences, got %d", len(expectedAud), len(aud))
	}
	for i, a := range expectedAud {
		if aud[i] != a {
			t.Errorf("audience[%d]: expected %q, got %q", i, a, aud[i])
		}
	}
}

func TestAccessToken_ContainsJTI(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, _ := svc.AccessToken("u", "o")
	parsed, _ := svc.parseToken(token)
	claims := parsed.Claims.(jwt.MapClaims)

	jti, ok := claims["jti"]
	if !ok || jti == "" {
		t.Error("expected non-empty jti claim")
	}
}

func TestAccessToken_UniquePerCall(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	t1, _ := svc.AccessToken("u", "o")
	t2, _ := svc.AccessToken("u", "o")
	if t1 == t2 {
		t.Error("expected unique tokens per call (different jti / kid)")
	}
}

func TestAccessToken_MissingPrivateKey(t *testing.T) {
	// Temporarily move the key file so it's missing
	privPath := string(PrivateKeyPath)
	origData, existed := backupFile(t, privPath)
	if existed {
		os.Remove(privPath)
		defer restoreFile(privPath, origData, true)
	}

	svc := newService(&mockRefreshTokenRepo{})
	_, err := svc.AccessToken("u", "o")
	if err == nil {
		t.Fatal("expected error when private key file is missing")
	}
}

// ===========================================================================
// RefreshToken tests
// ===========================================================================

func TestRefreshToken_Success(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{}
	svc := newService(mock)

	token, err := svc.RefreshToken("org-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if !mock.insertCalled {
		t.Error("expected Insert to be called on the repo")
	}
}

func TestRefreshToken_InsertsCorrectFields(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{}
	svc := newService(mock)

	_, err := svc.RefreshToken("org-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rt := mock.lastInserted
	if rt == nil {
		t.Fatal("expected a RefreshToken to be inserted")
	}
	if rt.ID == "" {
		t.Error("expected non-empty ID")
	}
	if rt.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %q", rt.UserID)
	}
	if rt.TokenHash == "" {
		t.Error("expected non-empty TokenHash")
	}
	if rt.ExpiresAt.Before(time.Now()) {
		t.Error("expected ExpiresAt in the future")
	}
}

func TestRefreshToken_RepoInsertError(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{
		insertErr: errors.New("db connection refused"),
	}
	svc := newService(mock)

	_, err := svc.RefreshToken("org-1", "user-1")
	if err == nil {
		t.Fatal("expected error when repo Insert fails")
	}
	if err.Error() != "db connection refused" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRefreshToken_TokenIsParseable(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, _ := svc.RefreshToken("org-1", "user-1")

	parsed, err := svc.parseToken(token)
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

func TestRefreshToken_ExpiryIs24Hours(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, _ := svc.RefreshToken("org-1", "user-1")

	parsed, _ := svc.parseToken(token)
	claims := parsed.Claims.(jwt.MapClaims)

	exp, _ := claims.GetExpirationTime()
	expectedExp := time.Now().Add(time.Duration(RefreshTokenExpiresAt) * time.Minute)
	diff := expectedExp.Sub(exp.Time)
	if diff < -30*time.Second || diff > 30*time.Second {
		t.Errorf("refresh expiry not within expected range: got %v, expected ~%v", exp.Time, expectedExp)
	}
}

// ===========================================================================
// toRefreshTokenModel tests
// ===========================================================================

func TestToRefreshTokenModel_FieldMapping(t *testing.T) {
	svc := newService(&mockRefreshTokenRepo{})
	rt, err := svc.toRefreshTokenModel("token-hash-value", "id-123", "user-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.TokenHash != "token-hash-value" {
		t.Errorf("expected TokenHash=token-hash-value, got %q", rt.TokenHash)
	}
	if rt.ID != "id-123" {
		t.Errorf("expected ID=id-123, got %q", rt.ID)
	}
	if rt.UserID != "user-456" {
		t.Errorf("expected UserID=user-456, got %q", rt.UserID)
	}
	if rt.ExpiresAt.Before(time.Now()) {
		t.Error("expected ExpiresAt in the future")
	}
}

// ===========================================================================
// parseToken tests
// ===========================================================================

func TestParseToken_ValidToken(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, _ := svc.AccessToken("user-1", "org-1")

	parsed, err := svc.parseToken(token)
	if err != nil {
		t.Fatalf("expected valid token to parse: %v", err)
	}
	if !parsed.Valid {
		t.Error("expected parsed token to be valid")
	}
}

func TestParseToken_TamperedToken(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, _ := svc.AccessToken("user-1", "org-1")

	// Tamper with the signature
	tampered := token[:len(token)-5] + "XXXXX"
	_, err := svc.parseToken(tampered)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestParseToken_CompletelyInvalidString(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	_, err := svc.parseToken("this.is.not.a.jwt")
	if err == nil {
		t.Fatal("expected error for invalid token string")
	}
}

func TestParseToken_EmptyString(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	_, err := svc.parseToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestParseToken_ExpiredToken(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})

	// Manually create an already-expired token
	privateKey, _ := svc.getPrivateKey(PrivateKeyPath)
	claims := jwt.RegisteredClaims{
		Subject:   "user-1",
		Issuer:    "org-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		ID:        "expired-id",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, _ := token.SignedString(privateKey)

	_, err := svc.parseToken(signed)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

// ===========================================================================
// validateRefreshToken tests
// ===========================================================================

func TestValidateRefreshToken_ValidToken(t *testing.T) {
	svc := newService(&mockRefreshTokenRepo{})
	rt := &RefreshToken{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := svc.validateRefreshToken(rt, "user-1")
	if err != nil {
		t.Fatalf("expected no error for valid refresh token: %v", err)
	}
}

func TestValidateRefreshToken_Expired(t *testing.T) {
	svc := newService(&mockRefreshTokenRepo{})
	rt := &RefreshToken{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	err := svc.validateRefreshToken(rt, "user-1")
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
	if err.Error() != "token is expired" {
		t.Errorf("expected 'token is expired', got %q", err.Error())
	}
}

func TestValidateRefreshToken_UserIDMismatch(t *testing.T) {
	svc := newService(&mockRefreshTokenRepo{})
	rt := &RefreshToken{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := svc.validateRefreshToken(rt, "user-999")
	if err == nil {
		t.Fatal("expected error for user ID mismatch")
	}
}

func TestValidateRefreshToken_ExpiredAndMismatch(t *testing.T) {
	svc := newService(&mockRefreshTokenRepo{})
	rt := &RefreshToken{
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	// Expired check comes first, so we should get expiry error
	err := svc.validateRefreshToken(rt, "user-999")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "token is expired" {
		t.Errorf("expected expiry error to take precedence, got %q", err.Error())
	}
}

// ===========================================================================
// ValidateToken (full refresh flow) tests
// ===========================================================================

func TestValidateToken_Success(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{}
	svc := newService(mock)

	// Generate a refresh token first
	refreshTokenStr, err := svc.RefreshToken("org-1", "user-1")
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	// The inserted record — set up mock to return it on FindByID
	insertedRecord := mock.lastInserted
	mock.findByIDResult = &RefreshToken{
		ID:        insertedRecord.ID,
		UserID:    "user-1",
		TokenHash: insertedRecord.TokenHash,
		ExpiresAt: insertedRecord.ExpiresAt,
	}

	resp, err := svc.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty new access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty new refresh token")
	}
	if !mock.updateCalled {
		t.Error("expected Update to be called on the repo")
	}
}

func TestValidateToken_InvalidTokenString(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	_, err := svc.ValidateRefreshToken("garbage-token")
	if err == nil {
		t.Fatal("expected error for invalid token string")
	}
}

func TestValidateToken_FindByIDError(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{
		findByIDErr: errors.New("db error"),
	}
	svc := newService(mock)

	refreshToken, _ := svc.RefreshToken("org-1", "user-1")

	_, err := svc.ValidateRefreshToken(refreshToken)
	if err == nil {
		t.Fatal("expected error when FindByID fails")
	}
	if err.Error() != "db error" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateToken_ExpiredRefreshInDB(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{}
	svc := newService(mock)

	refreshToken, _ := svc.RefreshToken("org-1", "user-1")

	// DB returns an expired record
	mock.findByIDResult = &RefreshToken{
		ID:        mock.lastInserted.ID,
		UserID:    "user-1",
		TokenHash: mock.lastInserted.TokenHash,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
	}

	_, err := svc.ValidateRefreshToken(refreshToken)
	if err == nil {
		t.Fatal("expected error for expired refresh token in DB")
	}
}

func TestValidateToken_UserIDMismatchInDB(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{}
	svc := newService(mock)

	refreshToken, _ := svc.RefreshToken("org-1", "user-1")

	// DB returns a record with a different user
	mock.findByIDResult = &RefreshToken{
		ID:        mock.lastInserted.ID,
		UserID:    "user-hacker",
		TokenHash: mock.lastInserted.TokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	_, err := svc.ValidateRefreshToken(refreshToken)
	if err == nil {
		t.Fatal("expected error for user ID mismatch")
	}
}

func TestValidateToken_UpdateError(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	mock := &mockRefreshTokenRepo{
		updateErr: errors.New("update failed"),
	}
	svc := newService(mock)

	refreshToken, _ := svc.RefreshToken("org-1", "user-1")

	mock.findByIDResult = &RefreshToken{
		ID:        mock.lastInserted.ID,
		UserID:    "user-1",
		TokenHash: mock.lastInserted.TokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	_, err := svc.ValidateRefreshToken(refreshToken)
	if err == nil {
		t.Fatal("expected error when repo Update fails")
	}
}

// ===========================================================================
// getPrivateKey / getPublickey tests
// ===========================================================================

func TestGetPrivateKey_Success(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	key, err := svc.getPrivateKey(PrivateKeyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil private key")
	}
}

func TestGetPrivateKey_FileNotFound(t *testing.T) {
	svc := newService(&mockRefreshTokenRepo{})
	_, err := svc.getPrivateKey("nonexistent/path.pem")
	if err == nil {
		t.Fatal("expected error when private key file is missing")
	}
}

func TestGetPublicKey_Success(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	key, err := svc.getPublickey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil public key")
	}
}

func TestGetPublicKey_FileNotFound(t *testing.T) {
	// Temporarily remove public key
	pubPath := string(PublicKeyPath)
	origData, existed := backupFile(t, pubPath)
	if existed {
		os.Remove(pubPath)
		defer restoreFile(pubPath, origData, true)
	}

	svc := newService(&mockRefreshTokenRepo{})
	_, err := svc.getPublickey()
	if err == nil {
		t.Fatal("expected error when public key file is missing")
	}
}

// ===========================================================================
// Signing method verification
// ===========================================================================

func TestToken_UsesES256(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, _ := svc.AccessToken("u", "o")

	parsed, _ := svc.parseToken(token)
	if parsed.Method.Alg() != "ES256" {
		t.Errorf("expected ES256 signing method, got %s", parsed.Method.Alg())
	}
}

func TestToken_ContainsKidHeader(t *testing.T) {
	cleanup := setupTestKeys(t)
	defer cleanup()

	svc := newService(&mockRefreshTokenRepo{})
	token, _ := svc.AccessToken("u", "o")

	parsed, _ := svc.parseToken(token)
	kid, ok := parsed.Header["kid"]
	if !ok {
		t.Fatal("expected kid in token header")
	}
	if kid == "" {
		t.Error("expected non-empty kid")
	}
}
