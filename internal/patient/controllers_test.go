package patient_test

import (
	"net/http"
	"testing"

	"hospital-backend/internal/patient"
	"hospital-backend/internal/patient/dto"
	"hospital-backend/internal/patient/mocks"
	"hospital-backend/internal/testutil/controllertest"
	errwrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

const validPatientBody = `{
	"name":"John Doe",
	"blood_group":"O+",
	"address":"123 Main St",
	"age":"30",
	"user_id":"user-1",
	"weight":"70",
	"gender":"male",
	"organisation_id":"org-1",
	"email_id":"john@example.com",
	"mobile_number":"9876543210"
}`

func TestAddGeneralInfoHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockPatientServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing name",
			body:       `{"blood_group":"O+","address":"a","age":"30","user_id":"u","weight":"70","gender":"m","organisation_id":"o","email_id":"e@x.com","mobile_number":"9876543210"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "patient already exists",
			body: validPatientBody,
			setup: func(m *mocks.MockPatientServicer) {
				m.EXPECT().CreatePatientSrv(gomock.Any(), gomock.Any()).Return("", errwrap.ErrPatientAlreadyExists)
			},
			wantStatus: fiber.StatusConflict,
		},
		{
			name: "success",
			body: validPatientBody,
			setup: func(m *mocks.MockPatientServicer) {
				m.EXPECT().CreatePatientSrv(gomock.Any(), gomock.Any()).Return("pat-1", nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockPatientServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			patientCtrl := patient.NewPatientControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/patients", patientCtrl.AddGeneralInfoHandler)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/patients",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetPatientByID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockPatientServicer)
		wantStatus int
	}{
		{
			name: "not found",
			path: "/patients/pat-1",
			setup: func(m *mocks.MockPatientServicer) {
				m.EXPECT().FindOne(gomock.Any(), gomock.Any()).Return(dto.PatientResponse{}, errwrap.ErrPatientNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			path: "/patients/pat-1",
			setup: func(m *mocks.MockPatientServicer) {
				m.EXPECT().FindOne(gomock.Any(), gomock.Any()).Return(dto.PatientResponse{PatientID: "pat-1", PatientName: "John"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockPatientServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			patientCtrl := patient.NewPatientControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/patients/:patientID", patientCtrl.GetPatientByID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindPatients(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockPatientServicer)
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
			setup: func(m *mocks.MockPatientServicer) {
				m.EXPECT().FindMany(gomock.Any(), gomock.Any()).Return([]dto.PatientResponse{{PatientID: "pat-1"}}, int64(1), nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockPatientServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			patientCtrl := patient.NewPatientControllerInterface(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/patients/list", patientCtrl.Find)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/patients/list",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
