package organisation_test

import (
	"net/http"
	"testing"

	"hospital-backend/internal/organisation"
	"hospital-backend/internal/organisation/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestCreateOrganisation(t *testing.T) {
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
			name:       "missing organisation_name",
			body:       `{"legal_entity_name":"LE","hospital_type":"general"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: `{"organisation_name":"City Hospital","legal_entity_name":"City Hospital LLC","hospital_type":"general"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().CreateOrganisation(gomock.Any(), gomock.Any()).Return("org-1", nil)
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
			orgCtrl := organisation.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/organisations", orgCtrl.CreateOrganisation)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/organisations",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestUpdateOrganisationLoc(t *testing.T) {
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
			body:       `{"state_id":"s1","city_id":"c1","country_id":"co1"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "organisation not found",
			body: `{"organisation_id":"org-1","state_id":"s1","city_id":"c1","country_id":"co1"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().UpdateOrganisationLoc(gomock.Any(), gomock.Any()).Return(wrapError.ErrOrganisationNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			body: `{"organisation_id":"org-1","state_id":"s1","city_id":"c1","country_id":"co1"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().UpdateOrganisationLoc(gomock.Any(), gomock.Any()).Return(nil)
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
			orgCtrl := organisation.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/organisations/location", orgCtrl.UpdateOrganisationLoc)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/organisations/location",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetByID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockOrganisationServicer)
		wantStatus int
	}{
		{
			name: "not found",
			path: "/organisations/org-1",
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().GetOrgByID(gomock.Any(), gomock.Any()).Return(organisation.Organisation{}, wrapError.ErrOrganisationNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			path: "/organisations/org-1",
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().GetOrgByID(gomock.Any(), gomock.Any()).Return(organisation.Organisation{ID: "org-1", OrganisationName: "City Hospital"}, nil)
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
			orgCtrl := organisation.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/organisations/:organisation_id", orgCtrl.GetByID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestUpdateOrganisation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockOrganisationServicer)
		wantStatus int
	}{
		{
			name:       "missing hospital_type",
			body:       `{"organisation_id":"org-1","organisation_name":"City","legal_entity_name":"City LLC"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: `{"organisation_id":"org-1","organisation_name":"City","legal_entity_name":"City LLC","hospital_type":"general"}`,
			setup: func(m *mocks.MockOrganisationServicer) {
				m.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
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
			orgCtrl := organisation.NewIOrganisationController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/organisations/update", orgCtrl.Update)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/organisations/update",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
