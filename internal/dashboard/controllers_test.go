package dashboard_test

import (
	"net/http"
	"testing"

	apptdto "hospital-backend/internal/appointments/dto"
	billingdto "hospital-backend/internal/billing/dto"
	empdto "hospital-backend/internal/employee/dto"
	rxdto "hospital-backend/internal/prescription/dto"
	"hospital-backend/internal/dashboard"
	"hospital-backend/internal/dashboard/mocks"
	"hospital-backend/internal/testutil/controllertest"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestGetAppointmentsGroupedByStatus(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockDashboardServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?organisation_id=org-1",
			setup: func(m *mocks.MockDashboardServicer) {
				m.EXPECT().GetAppointmentsGroupedByStatus(gomock.Any(), "org-1").
					Return(apptdto.AppointmentStatusCounts{}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockDashboardServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := dashboard.NewDashboardController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/dashboard/getByStatus", controller.GetAppointmentsGroupedByStatus)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/dashboard/getByStatus" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetTodayLatestAppointments(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockDashboardServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?organisation_id=org-1",
			setup: func(m *mocks.MockDashboardServicer) {
				m.EXPECT().GetTodayLatestAppointments(gomock.Any(), "org-1").
					Return([]apptdto.AppointmentList{}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockDashboardServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := dashboard.NewDashboardController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/dashboard/getTodayAppointments", controller.GetTodayLatestAppointments)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/dashboard/getTodayAppointments" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetTodayCompletedInvoiceSummary(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockDashboardServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?organisation_id=org-1",
			setup: func(m *mocks.MockDashboardServicer) {
				m.EXPECT().GetTodayCompletedInvoiceSummary(gomock.Any(), "org-1").
					Return(billingdto.TodayInvoiceCollectionSummary{}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockDashboardServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := dashboard.NewDashboardController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/dashboard/getTodayInvoiceSummary", controller.GetTodayCompletedInvoiceSummary)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/dashboard/getTodayInvoiceSummary" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetEmployeeStatusCounts(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockDashboardServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?organisation_id=org-1",
			setup: func(m *mocks.MockDashboardServicer) {
				m.EXPECT().GetEmployeeStatusCounts(gomock.Any(), "org-1").
					Return(empdto.EmployeeStatusCounts{Active: 2, Inactive: 1, Total: 3}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockDashboardServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := dashboard.NewDashboardController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/dashboard/getEmployeeStatusCounts", controller.GetEmployeeStatusCounts)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/dashboard/getEmployeeStatusCounts" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetTodayPrescriptions(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockDashboardServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?organisation_id=org-1",
			setup: func(m *mocks.MockDashboardServicer) {
				m.EXPECT().GetTodayPrescriptions(gomock.Any(), "org-1").
					Return(rxdto.TodayPrescriptionsSummary{}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockDashboardServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := dashboard.NewDashboardController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/dashboard/getTodayPrescriptions", controller.GetTodayPrescriptions)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/dashboard/getTodayPrescriptions" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
