package organisations_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"hospital-backend/central/organisations"
	"hospital-backend/central/organisations/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestControllerAddOrganisation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockOrganisationServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing tenant_id",
			body:       `{"legal_entity_name":"Acme","organisation_type":"hospital","facility_name":"Main"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing facility_name",
			body:       `{"tenant_id":"t1","legal_entity_name":"Acme","organisation_type":"hospital"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "whitespace only tenant_id",
			body:       `{"tenant_id":"   ","legal_entity_name":"Acme","organisation_type":"hospital","facility_name":"Main"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing organisation_type",
			body:       `{"tenant_id":"t1","legal_entity_name":"Acme","facility_name":"Main"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "invalid organisation_type typo",
			body: `{"tenant_id":"t1","legal_entity_name":"Acme","organisation_type":"hospial","facility_name":"Main"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().AddOrganisation(gomock.Any(), gomock.Any()).Return("", wrapError.ErrInvalidRequest)
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid license_expiry format",
			body:       `{"tenant_id":"t1","legal_entity_name":"Acme","organisation_type":"hospital","facility_name":"Main","license_expiry":"31-12-2027"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service invalid request",
			body: `{"tenant_id":"t1","legal_entity_name":"Acme","organisation_type":"hospital","facility_name":"Main"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().AddOrganisation(gomock.Any(), gomock.Any()).Return("", wrapError.ErrInvalidRequest)
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service create failed",
			body: `{"tenant_id":"t1","legal_entity_name":"Acme","organisation_type":"hospital","facility_name":"Main"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().AddOrganisation(gomock.Any(), gomock.Any()).Return("", wrapError.ErrOrganisationCreateFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			body: `{
				"tenant_id":"t1",
				"legal_entity_name":"Acme Health Pvt Ltd",
				"organisation_type":"hospital",
				"facility_name":"Main Campus",
				"patient_lookup":true,
				"lab_reports":false
			}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().AddOrganisation(gomock.Any(), gomock.Any()).Return("org-1", nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockOrganisationServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			orgCtrl := organisations.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/add", orgCtrl.AddOrganisation)
			})
			resp, body := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/add",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
			if tt.wantStatus == fiber.StatusOK {
				var out map[string]any
				if err := json.Unmarshal(body, &out); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if out["organisation_id"] != "org-1" {
					t.Fatalf("unexpected response: %s", body)
				}
			}
		})
	}
}

func TestControllerGetByID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockOrganisationServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			path:       "/getbyid/",
			wantStatus: fiber.StatusNotFound, // fiber route mismatch / empty param handling
		},
		{
			name: "not found",
			path: "/getbyid/org-missing",
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().GetOrgByID(gomock.Any(), "org-missing").Return(organisations.Organisation{}, wrapError.ErrOrganisationNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "fetch failed",
			path: "/getbyid/org-1",
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisations.Organisation{}, wrapError.ErrOrganisationFetchFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			path: "/getbyid/org-1",
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisations.Organisation{
					ID:              "org-1",
					LegalEntityName: "Acme",
				}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockOrganisationServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			orgCtrl := organisations.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/getbyid/:organisation_id", orgCtrl.GetByID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			// Empty param path "/getbyid/" may 404 from router before handler.
			if tt.name == "missing organisation_id" {
				if resp.StatusCode != fiber.StatusNotFound && resp.StatusCode != fiber.StatusBadRequest {
					t.Fatalf("status = %d, want 404 or 400", resp.StatusCode)
				}
				return
			}
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestControllerListByTenant(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockOrganisationServicer)
		wantStatus int
	}{
		{
			name: "fetch failed",
			path: "/listByTenant/tenant-1",
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().ListByTenantID(gomock.Any(), "tenant-1").Return(nil, wrapError.ErrOrganisationFetchFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			path: "/listByTenant/tenant-1",
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().ListByTenantID(gomock.Any(), "tenant-1").Return([]organisations.Organisation{
					{ID: "org-1"},
				}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockOrganisationServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			orgCtrl := organisations.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/listByTenant/:tenant_id", orgCtrl.ListByTenant)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestControllerUpdate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockOrganisationServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing organisation_id",
			body:       `{"facility_name":"New"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "invalid organisation_type",
			body: `{"organisation_id":"org-1","organisation_type":"nursing_home"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().Update(gomock.Any(), "org-1", gomock.Any()).Return(wrapError.ErrInvalidRequest)
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid license_expiry format",
			body:       `{"organisation_id":"org-1","facility_name":"New","license_expiry":"2027/12/31"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "not found",
			body: `{"organisation_id":"org-1","facility_name":"New"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().Update(gomock.Any(), "org-1", gomock.Any()).Return(wrapError.ErrOrganisationNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			body: `{"organisation_id":"org-1","facility_name":"New Campus","gstin":"29BBBBB0000B1Z5"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().Update(gomock.Any(), "org-1", gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockOrganisationServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			orgCtrl := organisations.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Patch("/update", orgCtrl.Update)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPatch,
				Path:   "/update",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestControllerUpdateAddress(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockOrganisationServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			body:       `{"country_id":"IN","state_id":"KA","city_id":"BLR"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing city_id",
			body:       `{"organisation_id":"org-1","country_id":"IN","state_id":"KA"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "wrong state key uses state instead of state_id",
			body:       `{"organisation_id":"org-1","country_id":"IN","state":"KA","city_id":"BLR"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "not found",
			body: `{"organisation_id":"org-1","country_id":"IN","state_id":"KA","city_id":"BLR"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().UpdateAddress(gomock.Any(), gomock.Any()).Return(wrapError.ErrOrganisationNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			body: `{
				"organisation_id":"org-1",
				"country_id":"IN",
				"state_id":"KA",
				"city_id":"BLR",
				"patient_lookup":true,
				"lab_reports":false
			}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().UpdateAddress(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockOrganisationServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			orgCtrl := organisations.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Patch("/updateAddress", orgCtrl.UpdateAddress)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPatch,
				Path:   "/updateAddress",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
