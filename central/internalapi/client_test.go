package internalapi_test

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-backend/central/customers"
	"hospital-backend/central/internalapi"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

type stubCustomers struct {
	customer customers.Customer
}

func (s stubCustomers) GetByTenantID(*zap.Logger, string) (customers.Customer, error) {
	return s.customer, nil
}

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

func TestProvisionHospitalAdmin(t *testing.T) {
	var gotPath string
	var gotBody struct {
		OrganisationID string `json:"organisation_id"`
		FirstName      string `json:"first_name"`
		Username       string `json:"username"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := internalapi.New(srv.URL, "id", "secret")
	client.Customers = stubCustomers{customer: customers.Customer{FullName: "Ada Lovelace", WorkEmail: "Ada@Example.com"}}
	if err := client.ProvisionHospitalAdmin(zap.NewNop(), "tenant-1", "org-1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotPath != "/api/v1/hospital/internal/organisation/addFirstUser" || gotBody.FirstName != "Ada" || gotBody.Username != "ada" || gotBody.OrganisationID != "org-1" {
		t.Fatalf("path=%s body=%+v", gotPath, gotBody)
	}
}
