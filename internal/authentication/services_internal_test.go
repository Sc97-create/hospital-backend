package authentication

import (
	"testing"

	"hospital-backend/internal/authentication/dto"
	"hospital-backend/internal/jwt"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"
)

func TestValidateCredentials(t *testing.T) {
	svc := NewService(nil, nil, nil)

	if err := svc.validateCredentials(dto.LoginUser{}); err == nil {
		t.Fatal("expected error for empty credentials")
	}
	if err := svc.validateCredentials(dto.LoginUser{Username: "u", Password: "p"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateNewPassword(t *testing.T) {
	svc := NewService(nil, nil, nil)

	tests := []struct {
		name string
		req  dto.UpdatePasswordRequest
	}{
		{name: "empty", req: dto.UpdatePasswordRequest{}},
		{name: "mismatch", req: dto.UpdatePasswordRequest{Password: "abc", ConfirmPassword: "xyz"}},
		{name: "too short", req: dto.UpdatePasswordRequest{Password: "short", ConfirmPassword: "short"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.validateNewPassword(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	if err := svc.validateNewPassword(servicetest.ValidUpdatePasswordRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompareAndHashPassword(t *testing.T) {
	svc := NewService(nil, nil, nil)
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
	svc := NewService(nil, nil, nil)
	resp := svc.toLoginResp(jwt.TokenResp{AccessToken: "access", RefreshToken: "refresh"})
	if resp.Token != "access" || resp.RefreshToken != "refresh" {
		t.Fatalf("unexpected mapping: %+v", resp)
	}
}

func TestUpdatePasswordValidationOnly(t *testing.T) {
	svc := NewService(nil, nil, nil)
	err := svc.UpdatePassword(servicetest.NopLogger(), "user-1", dto.UpdatePasswordRequest{Password: "x", ConfirmPassword: "y"})
	if err != wrapError.ErrInvalidRequest {
		t.Fatalf("expected invalid request, got %v", err)
	}
}
