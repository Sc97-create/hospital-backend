package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-backend/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHealth(t *testing.T) {
	if logger.Log == nil {
		logger.Init("development", "error")
	}

	db, err := gorm.Open(sqlite.Open("file:health?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	okApp := fiber.New()
	RegisterHealth(okApp, db)
	resp := doHealth(t, okApp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if body.Status != "ok" {
		t.Fatalf("status=%q", body.Status)
	}

	down := fiber.New()
	RegisterHealth(down, nil)
	resp = doHealth(t, down)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func doHealth(t *testing.T, app *fiber.App) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestHealthPublicRoute(t *testing.T) {
	if !IsPublicRoute("/api/v1/health") {
		t.Fatal("health should be public")
	}
}
