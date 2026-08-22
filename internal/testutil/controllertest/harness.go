package controllertest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-backend/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const loggerKey = "req_logger"
const userIDKey = "user_id"

func init() {
	if logger.Log == nil {
		logger.Init("development", "error")
	}
}

// NewApp builds a Fiber app with a request-scoped logger for controller tests.
func NewApp(t *testing.T, register func(app *fiber.App)) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(attachTestLogger())
	register(app)
	return app
}

func attachTestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(loggerKey, logger.Log)
		return c.Next()
	}
}

// WithUserID sets the authenticated user on a Fiber context.
func WithUserID(userID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(userIDKey, userID)
		return c.Next()
	}
}

// Request describes an HTTP call against a Fiber app in tests.
type Request struct {
	Method  string
	Path    string
	Body    any
	Headers map[string]string
	Cookies map[string]string
}

// Do executes req against app and returns the HTTP response and raw body.
func Do(t *testing.T, app *fiber.App, req Request) (*http.Response, []byte) {
	t.Helper()
	if req.Method == "" {
		req.Method = http.MethodPost
	}

	var bodyReader io.Reader
	switch b := req.Body.(type) {
	case nil:
	case string:
		bodyReader = bytes.NewBufferString(b)
	case []byte:
		bodyReader = bytes.NewBuffer(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewBuffer(raw)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.Cookies {
		httpReq.AddCookie(&http.Cookie{Name: k, Value: v})
	}

	resp, err := app.Test(httpReq, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, raw
}

// AssertStatus fails the test when the response status does not match want.
func AssertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("status = %d, want %d", resp.StatusCode, want)
	}
}

// TestLogger returns the global zap logger used in controller tests.
func TestLogger() *zap.Logger {
	return logger.Log
}
