package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hospital-backend/config"
	"hospital-backend/internal/jwt"
	"hospital-backend/internal/modules"
	rpdto "hospital-backend/internal/rolepermissions/dto"
	"hospital-backend/pkg/logger"

	"github.com/gofiber/fiber/v2"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockRoleLookup struct {
	byUser map[string]string
	err    error
}

func (m *mockRoleLookup) FindRoleIDByUserID(userID string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	roleID, ok := m.byUser[userID]
	if !ok {
		return "", errors.New("user not found")
	}
	return roleID, nil
}

type mockRoleAccessLoader struct {
	byRole map[string]rpdto.RoleAccess
	err    error
}

func (m *mockRoleAccessLoader) FindModulesByRoleID(roleID string) (rpdto.RoleAccess, error) {
	if m.err != nil {
		return rpdto.RoleAccess{}, m.err
	}
	access, ok := m.byRole[roleID]
	if !ok {
		return rpdto.RoleAccess{Permissions: []rpdto.RoleModulePermission{}}, nil
	}
	return access, nil
}

type stubRefreshRepo struct{}

func (stubRefreshRepo) Insert(*jwt.RefreshToken) error             { return nil }
func (stubRefreshRepo) FindByID(string) (*jwt.RefreshToken, error) { return nil, nil }
func (stubRefreshRepo) Update(string, time.Time, string) error     { return nil }
func (stubRefreshRepo) CheckIfExist(string) (int64, error)         { return 0, nil }
func (stubRefreshRepo) FindIDByUserID(string) (string, error)      { return "", nil }
func (stubRefreshRepo) DeleteByID(string) error                    { return nil }
func (stubRefreshRepo) DeleteByUserID(string) error                { return nil }

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

const (
	userViewer  = "user-viewer"
	userCreator = "user-creator"
	userAdmin   = "user-admin"
	userNone    = "user-none"

	roleViewer  = "role-viewer"
	roleCreator = "role-creator"
	roleAdmin   = "role-admin"
	roleNone    = "role-none"
)

func init() {
	if logger.Log == nil {
		logger.Init("development", "error")
	}
}

func setupRBACKeys(t *testing.T) (privPath, pubPath string) {
	t.Helper()
	// JwtService joins paths with os.Getwd(); keep keys relative to the package dir.
	dir := filepath.Join("testdata", "rbac_keys", strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir keys: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join("testdata", "rbac_keys")) })

	privPath = filepath.Join(dir, "private.pem")
	pubPath = filepath.Join(dir, "public.pem")

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private: %v", err)
	}
	if err := os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}), 0o600); err != nil {
		t.Fatalf("write private: %v", err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public: %v", err)
	}
	if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}), 0o644); err != nil {
		t.Fatalf("write public: %v", err)
	}
	return privPath, pubPath
}

func newTestJWT(t *testing.T) *jwt.JwtService {
	t.Helper()
	svc, _, _ := newTestJWTWithKeys(t)
	return svc
}

func newTestJWTWithKeys(t *testing.T) (svc *jwt.JwtService, privPath, pubPath string) {
	t.Helper()
	privPath, pubPath = setupRBACKeys(t)
	cfg := &config.Config{PrivateKeyPath: privPath, PublicKeyPath: pubPath}
	return jwt.NewJwtService(stubRefreshRepo{}, cfg), privPath, pubPath
}

func defaultRBACFixture() (*mockRoleLookup, *mockRoleAccessLoader) {
	lookup := &mockRoleLookup{byUser: map[string]string{
		userViewer:  roleViewer,
		userCreator: roleCreator,
		userAdmin:   roleAdmin,
		userNone:    roleNone,
	}}
	loader := &mockRoleAccessLoader{byRole: map[string]rpdto.RoleAccess{
		roleViewer: {
			Permissions: []rpdto.RoleModulePermission{{
				ModuleName:  modules.Patient,
				Permissions: rpdto.ModulePermissionFlags{View: true},
			}},
		},
		roleCreator: {
			Permissions: []rpdto.RoleModulePermission{{
				ModuleName:  modules.Employee,
				Permissions: rpdto.ModulePermissionFlags{Create: true},
			}},
		},
		roleAdmin: {IsAdmin: true},
		roleNone:  {Permissions: []rpdto.RoleModulePermission{}},
	}}
	return lookup, loader
}

func newRBACTestApp(t *testing.T, jwtSvc *jwt.JwtService, loader RoleAccessLoader, lookup RoleIDFinder) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(RequestLogger())

	api := app.Group("/api/v1")

	patients := api.Group("/patients")
	UseProtected(patients, jwtSvc, loader, lookup)
	patients.Post("/getPatients", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true, "route": "getPatients"})
	})
	patients.Post("/addGeneralInfo", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true, "route": "addGeneralInfo"})
	})
	patients.Get("/getpatientByID/:patientID", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true, "id": c.Params("patientID")})
	})

	employees := api.Group("/employee")
	UseProtected(employees, jwtSvc, loader, lookup)
	employees.Post("/create", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true, "route": "create"})
	})
	employees.Delete("/delete", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true, "route": "delete"})
	})

	// Unmapped authenticated route — should fail closed when RBAC is attached.
	unmapped := api.Group("/bed")
	UseProtected(unmapped, jwtSvc, loader, lookup)
	unmapped.Post("/createBed", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
	})

	// Public login (no RBAC middleware) — control case.
	auth := api.Group("/authentication")
	auth.Post("/login", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true, "route": "login"})
	})

	return app
}

func issueToken(t *testing.T, jwtSvc *jwt.JwtService, userID string) string {
	t.Helper()
	token, err := jwtSvc.AccessToken(userID, "org-test")
	if err != nil {
		t.Fatalf("AccessToken(%s): %v", userID, err)
	}
	return token
}

func doRequest(t *testing.T, app *fiber.App, method, path, bearer string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

func TestRBAC_E2E_Scenarios(t *testing.T) {
	jwtSvc := newTestJWT(t)
	lookup, loader := defaultRBACFixture()
	app := newRBACTestApp(t, jwtSvc, loader, lookup)

	viewerTok := issueToken(t, jwtSvc, userViewer)
	creatorTok := issueToken(t, jwtSvc, userCreator)
	adminTok := issueToken(t, jwtSvc, userAdmin)
	noneTok := issueToken(t, jwtSvc, userNone)

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		authHeader string
		wantStatus int
	}{
		{
			name:       "public login without token allowed",
			method:     http.MethodPost,
			path:       "/api/v1/authentication/login",
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "protected route without token → 401",
			method:     http.MethodPost,
			path:       "/api/v1/patients/getPatients",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "invalid auth header format → 401",
			method:     http.MethodPost,
			path:       "/api/v1/patients/getPatients",
			authHeader: "Token abc",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "invalid jwt → 401",
			method:     http.MethodPost,
			path:       "/api/v1/patients/getPatients",
			token:      "not-a-jwt",
			wantStatus: fiber.StatusUnauthorized,
		},
		{
			name:       "POST getPatients is view (not create) — viewer allowed",
			method:     http.MethodPost,
			path:       "/api/v1/patients/getPatients",
			token:      viewerTok,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "GET patient by id is view — viewer allowed",
			method:     http.MethodGet,
			path:       "/api/v1/patients/getpatientByID/p-1",
			token:      viewerTok,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "viewer cannot create patient",
			method:     http.MethodPost,
			path:       "/api/v1/patients/addGeneralInfo",
			token:      viewerTok,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "viewer cannot create employee",
			method:     http.MethodPost,
			path:       "/api/v1/employee/create",
			token:      viewerTok,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "creator can create employee",
			method:     http.MethodPost,
			path:       "/api/v1/employee/create",
			token:      creatorTok,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "creator cannot delete employee",
			method:     http.MethodDelete,
			path:       "/api/v1/employee/delete",
			token:      creatorTok,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "creator cannot view patients (wrong module)",
			method:     http.MethodPost,
			path:       "/api/v1/patients/getPatients",
			token:      creatorTok,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "admin bypasses module checks — create employee",
			method:     http.MethodPost,
			path:       "/api/v1/employee/create",
			token:      adminTok,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "admin bypasses — view patients",
			method:     http.MethodPost,
			path:       "/api/v1/patients/getPatients",
			token:      adminTok,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "admin bypasses — delete employee",
			method:     http.MethodDelete,
			path:       "/api/v1/employee/delete",
			token:      adminTok,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "role with no permissions denied on mapped route",
			method:     http.MethodPost,
			path:       "/api/v1/patients/getPatients",
			token:      noneTok,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "admin allowed on unmapped route (full access)",
			method:     http.MethodPost,
			path:       "/api/v1/bed/createBed",
			token:      adminTok,
			wantStatus: fiber.StatusOK,
		},
		{
			name:       "non-admin denied on unmapped route",
			method:     http.MethodPost,
			path:       "/api/v1/bed/createBed",
			token:      viewerTok,
			wantStatus: fiber.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			switch {
			case tt.authHeader != "":
				req.Header.Set("Authorization", tt.authHeader)
			case tt.token != "":
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", resp.StatusCode, tt.wantStatus, readBody(t, resp))
			}
		})
	}
}

func TestRBAC_E2E_RoleLookupFailure(t *testing.T) {
	jwtSvc := newTestJWT(t)
	lookup := &mockRoleLookup{err: errors.New("db down")}
	loader := &mockRoleAccessLoader{byRole: map[string]rpdto.RoleAccess{}}
	app := newRBACTestApp(t, jwtSvc, loader, lookup)

	tok := issueToken(t, jwtSvc, userViewer)
	resp := doRequest(t, app, http.MethodPost, "/api/v1/patients/getPatients", tok)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status=%d want=403 body=%s", resp.StatusCode, readBody(t, resp))
	}
}

func TestRBAC_E2E_PermissionLoadFailure(t *testing.T) {
	jwtSvc := newTestJWT(t)
	lookup := &mockRoleLookup{byUser: map[string]string{userViewer: roleViewer}}
	loader := &mockRoleAccessLoader{err: errors.New("permissions query failed")}
	app := newRBACTestApp(t, jwtSvc, loader, lookup)

	tok := issueToken(t, jwtSvc, userViewer)
	resp := doRequest(t, app, http.MethodPost, "/api/v1/patients/getPatients", tok)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status=%d want=403 body=%s", resp.StatusCode, readBody(t, resp))
	}
}

// TestRBAC_E2E_BypassAttempts covers privilege-escalation / auth-bypass attempts.
// Permissions are always derived from JWT sub → userID → role_id in DB — never from
// client-supplied role claims or headers.
func TestRBAC_E2E_BypassAttempts(t *testing.T) {
	jwtSvc, privPath, _ := newTestJWTWithKeys(t)
	lookup, loader := defaultRBACFixture()
	app := newRBACTestApp(t, jwtSvc, loader, lookup)

	viewerTok := issueToken(t, jwtSvc, userViewer)
	privKey := mustLoadECPrivateKey(t, privPath)

	t.Run("empty bearer token → 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/patients/getPatients", nil)
		req.Header.Set("Authorization", "Bearer ")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d want=401 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})

	t.Run("missing Bearer scheme → 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/patients/getPatients", nil)
		req.Header.Set("Authorization", viewerTok)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d want=401 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})

	t.Run("token signed with attacker key → 401", func(t *testing.T) {
		attackerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		tok := signClaims(t, attackerKey, jwtlib.RegisteredClaims{
			Subject:   userAdmin,
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		})
		resp := doRequest(t, app, http.MethodPost, "/api/v1/employee/create", tok)
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d want=401 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})

	t.Run("tampered payload after sign → 401", func(t *testing.T) {
		tok := issueToken(t, jwtSvc, userViewer)
		parts := strings.Split(tok, ".")
		if len(parts) != 3 {
			t.Fatalf("expected 3 jwt parts, got %d", len(parts))
		}
		// Corrupt payload while keeping header+signature shape.
		parts[1] = parts[1] + "tamper"
		resp := doRequest(t, app, http.MethodPost, "/api/v1/patients/getPatients", strings.Join(parts, "."))
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d want=401 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})

	t.Run("expired token → 401", func(t *testing.T) {
		tok := signClaims(t, privKey, jwtlib.RegisteredClaims{
			Subject:   userAdmin,
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(-time.Hour)),
		})
		resp := doRequest(t, app, http.MethodPost, "/api/v1/employee/create", tok)
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d want=401 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})

	t.Run("valid token for unknown userID → 403", func(t *testing.T) {
		tok := issueToken(t, jwtSvc, "user-does-not-exist")
		resp := doRequest(t, app, http.MethodPost, "/api/v1/patients/getPatients", tok)
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status=%d want=403 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})

	t.Run("viewer token cannot escalate via spoofed X-User-ID / X-Role-ID headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/employee/create", nil)
		req.Header.Set("Authorization", "Bearer "+viewerTok)
		req.Header.Set("X-User-ID", userAdmin)
		req.Header.Set("X-Role-ID", roleAdmin)
		req.Header.Set("X-Is-Admin", "true")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status=%d want=403 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})

	t.Run("permissions bound to JWT sub userID not swapped mid-request", func(t *testing.T) {
		// Creator may create employee; viewer may not. Same path, different userIDs.
		creatorTok := issueToken(t, jwtSvc, userCreator)
		respCreator := doRequest(t, app, http.MethodPost, "/api/v1/employee/create", creatorTok)
		if respCreator.StatusCode != fiber.StatusOK {
			t.Fatalf("creator status=%d want=200 body=%s", respCreator.StatusCode, readBody(t, respCreator))
		}
		respViewer := doRequest(t, app, http.MethodPost, "/api/v1/employee/create", viewerTok)
		if respViewer.StatusCode != fiber.StatusForbidden {
			t.Fatalf("viewer status=%d want=403 body=%s", respViewer.StatusCode, readBody(t, respViewer))
		}
	})

	t.Run("token with empty subject → 401", func(t *testing.T) {
		tok := signClaims(t, privKey, jwtlib.RegisteredClaims{
			Subject:   "",
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		})
		resp := doRequest(t, app, http.MethodPost, "/api/v1/patients/getPatients", tok)
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d want=401 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})

	t.Run("alg none style unsigned token → 401", func(t *testing.T) {
		// Classic bypass attempt: header.payload. with no valid signature.
		resp := doRequest(t, app, http.MethodPost, "/api/v1/patients/getPatients",
			"eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ1c2VyLWFkbWluIn0.")
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("status=%d want=401 body=%s", resp.StatusCode, readBody(t, resp))
		}
	})
}

func mustLoadECPrivateKey(t *testing.T, path string) *ecdsa.PrivateKey {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, path))
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		t.Fatal("decode pem failed")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}
	return key
}

func signClaims(t *testing.T, key *ecdsa.PrivateKey, claims jwtlib.RegisteredClaims) string {
	t.Helper()
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodES256, claims)
	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}
