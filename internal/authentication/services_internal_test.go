package authentication

import (
	"testing"

	"hospital-backend/internal/authentication/dto"
	"hospital-backend/internal/jwt"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"
	"net/url"
)

func TestValidateCredentials(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)

	if err := svc.validateCredentials(dto.LoginUser{}); err == nil {
		t.Fatal("expected error for empty credentials")
	}
	if err := svc.validateCredentials(dto.LoginUser{Username: "u", Password: "p"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateNewPassword(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)

	tests := []struct {
		name    string
		pass    string
		confirm string
	}{
		{name: "empty"},
		{name: "mismatch", pass: "abc", confirm: "xyz"},
		{name: "too short", pass: "short", confirm: "short"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.validateNewPassword(tt.pass, tt.confirm); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	req := servicetest.ValidUpdatePasswordRequest()
	if err := svc.validateNewPassword(req.Password, req.ConfirmPassword); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompareAndHashPassword(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)
	plain := "password123"

	hash, err := svc.hashPassword(plain)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ok, err := svc.comparePwd(hash, plain)
	if err != nil || !ok {
		t.Fatalf("compare failed: ok=%v err=%v", ok, err)
	}

	ok, err = svc.comparePwd(hash, "wrong")
	if err == nil || ok {
		t.Fatal("expected compare failure for wrong password")
	}
}

func TestToLoginResp(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)
	resp := svc.toLoginResp(jwt.TokenResp{AccessToken: "access", RefreshToken: "refresh"})
	if resp.Token != "access" || resp.RefreshToken != "refresh" {
		t.Fatalf("unexpected mapping: %+v", resp)
	}
}

func TestUpdatePasswordValidationOnly(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil)
	err := svc.UpdatePassword(servicetest.NopLogger(), dto.UpdatePasswordRequest{Password: "x", ConfirmPassword: "y"})
	if err != wrapError.ErrInvalidRequest {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestNormalizeResetToken(t *testing.T) {
	plain := "abc123def456"
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: plain, want: plain},
		{name: "trim", input: "  " + plain + "  ", want: plain},
		{name: "url encoded", input: url.QueryEscape(plain), want: plain},
		{name: "full url", input: "http://localhost:5173/forgot-password/reset?token=" + plain, want: plain},
		{name: "empty", input: "   ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeResetToken(tt.input); got != tt.want {
				t.Fatalf("normalizeResetToken(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreatePasswordResetTokenRoundTrip(t *testing.T) {
	plain, storedHash, err := createPasswordResetToken()
	if err != nil {
		t.Fatalf("createPasswordResetToken: %v", err)
	}
	if hashPasswordResetToken(plain) != storedHash {
		t.Fatalf("stored hash does not match hash(plain token)")
	}
}
