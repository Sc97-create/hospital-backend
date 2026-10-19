package internalapi_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-backend/central/internalapi"
	wrapError "hospital-backend/shared/error"
)

func TestAccessToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/v1/hospital/internal/jwt/accessToken" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"access_token":"tok-1"}`))
	}))
	defer srv.Close()

	got, err := internalapi.New(srv.URL, "id", "secret").AccessToken("user-1", "org-1")
	if err != nil || got != "tok-1" {
		t.Fatalf("token=%q err=%v", got, err)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("id:secret"))
	if gotAuth != wantAuth {
		t.Fatalf("auth=%q", gotAuth)
	}
}

func TestAccessTokenFailure(t *testing.T) {
	_, err := internalapi.New("", "id", "secret").AccessToken("user-1", "")
	if err != wrapError.ErrAccessTokenCreateFailed {
		t.Fatalf("err=%v", err)
	}
}
