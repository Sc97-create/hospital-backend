package centralapi_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-backend/internal/centralapi"
)

func TestCheckSubscriptionSendsBasicAuth(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"trial period ended","ended":true}`))
	}))
	defer srv.Close()

	ended, message, err := centralapi.New(srv.URL, "id", "secret").CheckSubscription(context.Background(), "org-1")
	if err != nil || !ended || message != "trial period ended" {
		t.Fatalf("ended=%v message=%q err=%v", ended, message, err)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("id:secret"))
	if gotAuth != wantAuth || gotPath != "/api/v1/central/internal/subscription/checkEnd/org-1" {
		t.Fatalf("auth=%q path=%s", gotAuth, gotPath)
	}
}
