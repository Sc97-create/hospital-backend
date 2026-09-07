package appointments_test

import (
	"net/http"
	"testing"

	"hospital-backend/internal/appointments"
	"hospital-backend/internal/appointments/dto"
	"hospital-backend/internal/appointments/mocks"
	"hospital-backend/internal/testutil/controllertest"
	errWrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

const validCreateAppointmentBody = `{
	"patient_id":"pat-1",
	"user_id":"user-1",
	"organisation_id":"org-1",
	"doctor_id":"doc-1",
	"start_time":"09:00",
	"end_time":"09:30",
	"appointment_date":"2026-08-22",
	"visit_type":"consultation"
}`

func TestCreateAppointment(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockAppointmentServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing patient_id",
			body:       `{"user_id":"u1","organisation_id":"o1","doctor_id":"d1","start_time":"09:00","end_time":"09:30","appointment_date":"2026-08-22","visit_type":"consultation"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "org schedule not found",
			body: validCreateAppointmentBody,
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().CreateApptmnt(gomock.Any(), gomock.Any()).
					Return(dto.NewApptmntResp{}, errWrap.ErrOrgScheduleNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			body: validCreateAppointmentBody,
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().CreateApptmnt(gomock.Any(), gomock.Any()).
					Return(dto.NewApptmntResp{ID: "appt-1", Code: "200"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAppointmentServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := appointments.NewAppointmentController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/appointments", controller.CreateAppointment)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/appointments",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetSlots(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockAppointmentServicer)
		wantStatus int
	}{
		{
			name:       "missing doctor_id",
			query:      "?organisation_id=org-1",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing organisation_id",
			query:      "?doctor_id=doc-1",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "org schedule not found",
			query: "?doctor_id=doc-1&organisation_id=org-1&date=2026-08-22",
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().GetSlots(gomock.Any(), "doc-1", "org-1", "2026-08-22").
					Return(dto.SlotResponse{}, errWrap.ErrOrgScheduleNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:  "success",
			query: "?doctor_id=doc-1&organisation_id=org-1&date=2026-08-22",
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().GetSlots(gomock.Any(), "doc-1", "org-1", "2026-08-22").
					Return(dto.SlotResponse{Code: "200"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAppointmentServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := appointments.NewAppointmentController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/appointments/slots", controller.GetSlots)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/appointments/slots" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindManyByOrganisationID(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockAppointmentServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing organisation_id",
			body:       `{"limit":10,"page_no":1}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: `{"organisation_id":"org-1","limit":10,"page_no":1}`,
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().GetAppointmentsByOrgID(gomock.Any(), gomock.Any()).
					Return([]dto.AppointmentList{{AppointmentID: "appt-1"}}, 1, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAppointmentServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := appointments.NewAppointmentController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/appointments/list", controller.FindManyByOrganisationID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/appointments/list",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindAppointmentsPreview(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockAppointmentServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			query:      "?appointment_id=appt-1",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing appointment_id",
			query:      "?organisation_id=org-1",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "not found",
			query: "?organisation_id=org-1&appointment_id=appt-1",
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().GetAppointmentPreview(gomock.Any(), "org-1", "appt-1").
					Return(dto.AppointmentDetails{}, errWrap.ErrAppointmentNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:  "success",
			query: "?organisation_id=org-1&appointment_id=appt-1",
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().GetAppointmentPreview(gomock.Any(), "org-1", "appt-1").
					Return(dto.AppointmentDetails{AppointmentID: "appt-1"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAppointmentServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := appointments.NewAppointmentController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/appointments/preview", controller.FindAppointmentsPreview)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/appointments/preview" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestUpdateAppointmentStatus(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockAppointmentServicer)
		wantStatus int
	}{
		{
			name:       "missing appointment_id",
			body:       `{"status":"completed"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "invalid status",
			body: `{"appointment_id":"appt-1","status":"bad"}`,
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(errWrap.ErrInvalidRequest)
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: `{"appointment_id":"appt-1","status":"completed"}`,
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAppointmentServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := appointments.NewAppointmentController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/appointments/status", controller.UpdateStatus)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/appointments/status",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetAppointmentByPatientID(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockAppointmentServicer)
		wantStatus int
	}{
		{
			name:       "missing patient_id",
			body:       `{"organisation_id":"org-1","limit":10,"page_no":1}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: `{"patient_id":"pat-1","organisation_id":"org-1","limit":10,"page_no":1}`,
			setup: func(m *mocks.MockAppointmentServicer) {
				m.EXPECT().GetAppointmentByPatientID(gomock.Any(), gomock.Any()).
					Return(dto.Response{Code: "200"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockAppointmentServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := appointments.NewAppointmentController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/appointments/patient", controller.GetAppointmentByPatientID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/appointments/patient",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
