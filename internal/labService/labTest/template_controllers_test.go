package labservice_test

import (
	"net/http"
	"testing"

	labservice "hospital-backend/internal/labService/labTest"
	"hospital-backend/internal/labService/labTest/dto"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type stubTemplateService struct {
	defaultsResp dto.DefaultsListResponse
	defaultsErr  error
	createID     string
	createErr    error
	getResp      dto.LabTestTemplateResponse
	getErr       error
	listRows     []dto.LabTestTemplateResponse
	listTotal    int64
	listErr      error
	lastOrgID    string
}

func (s *stubTemplateService) ListDefaults(_ *zap.Logger, organisationID string) (dto.DefaultsListResponse, error) {
	s.lastOrgID = organisationID
	return s.defaultsResp, s.defaultsErr
}

func (s *stubTemplateService) Create(_ *zap.Logger, _ dto.CreateLabTestTemplateRequest) (string, error) {
	return s.createID, s.createErr
}

func (s *stubTemplateService) GetByID(_ *zap.Logger, _ string) (dto.LabTestTemplateResponse, error) {
	return s.getResp, s.getErr
}

func (s *stubTemplateService) List(_ *zap.Logger, _ dto.ListLabTestTemplatesRequest) ([]dto.LabTestTemplateResponse, int64, error) {
	return s.listRows, s.listTotal, s.listErr
}

func TestLabTestTemplateControllerListDefaults(t *testing.T) {
	svc := &stubTemplateService{
		defaultsResp: dto.DefaultsListResponse{
			Defaults:  []dto.LabTemplateResponse{{TestCode: "HEM-01", TestName: "CBC"}},
			Templates: []dto.LabTestTemplateResponse{{ID: "tpl-1", Name: "Panel"}},
		},
	}
	ctrl := labservice.NewLabTestTemplateController(svc)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/lab-test-templates/defaults", ctrl.ListDefaults)
	})

	missingOrg, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-test-templates/defaults",
	})
	controllertest.AssertStatus(t, missingOrg, fiber.StatusBadRequest)

	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-test-templates/defaults?organisation_id=org-1",
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
	if svc.lastOrgID != "org-1" {
		t.Fatalf("expected organisation_id org-1, got %q", svc.lastOrgID)
	}
}

func TestLabTestTemplateControllerCreate(t *testing.T) {
	svc := &stubTemplateService{createID: "tpl-1"}
	ctrl := labservice.NewLabTestTemplateController(svc)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/lab-test-templates/create", ctrl.Create)
	})

	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/lab-test-templates/create",
		Body: `{
			"organisation_id":"org-1",
			"created_by":"user-1",
			"name":"Diabetes Panel",
			"test_codes":["BIO-04","BIO-12"]
		}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)

	bad, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/lab-test-templates/create",
		Body:   `{"organisation_id":"org-1"}`,
	})
	controllertest.AssertStatus(t, bad, fiber.StatusBadRequest)
}

func TestLabTestTemplateControllerGetAndList(t *testing.T) {
	svc := &stubTemplateService{
		getErr:    wrapError.ErrLabTestTemplateNotFound,
		listRows:  []dto.LabTestTemplateResponse{{ID: "tpl-1"}},
		listTotal: 1,
	}
	ctrl := labservice.NewLabTestTemplateController(svc)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/lab-test-templates/list", ctrl.List)
		app.Get("/lab-test-templates/getbyID/:id", ctrl.GetByID)
	})

	missing, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-test-templates/getbyID/missing",
	})
	controllertest.AssertStatus(t, missing, fiber.StatusNotFound)

	listBad, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-test-templates/list",
	})
	controllertest.AssertStatus(t, listBad, fiber.StatusBadRequest)

	listOK, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-test-templates/list?organisation_id=org-1",
	})
	controllertest.AssertStatus(t, listOK, fiber.StatusOK)
}
