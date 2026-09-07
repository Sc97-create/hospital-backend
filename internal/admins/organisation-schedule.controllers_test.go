package admins_test

import (
	"net/http"
	"testing"

	"hospital-backend/internal/admins"
	"hospital-backend/internal/admins/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

const validOrgScheduleBody = `{
	"organisation_id":"org-1",
	"start_time":"09:00",
	"end_time":"17:00",
	"time_slot":30,
	"break_start_time":"13:00",
	"break_end_time":"14:00",
	"week_offs":["SUN","SAT"],
	"is_closed":false
}`

func TestCreateOrgSchedule(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockOrganisationScheduleServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing organisation_id",
			body:       `{"start_time":"09:00","end_time":"17:00","time_slot":30,"break_start_time":"13:00","break_end_time":"14:00","week_offs":[]}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "invalid time_slot",
			body:       `{"organisation_id":"org-1","start_time":"09:00","end_time":"17:00","time_slot":0,"break_start_time":"13:00","break_end_time":"14:00","week_offs":[]}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service invalid request",
			body: validOrgScheduleBody,
			setup: func(m *mocks.MockOrganisationScheduleServicer) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(wrapError.ErrInvalidRequest)
			},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service failure",
			body: validOrgScheduleBody,
			setup: func(m *mocks.MockOrganisationScheduleServicer) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(wrapError.ErrOrgScheduleCreateFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			body: validOrgScheduleBody,
			setup: func(m *mocks.MockOrganisationScheduleServicer) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockOrganisationScheduleServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			controller := admins.NewOrgSchedController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/admins/org-schedule", controller.Create)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/admins/org-schedule",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
