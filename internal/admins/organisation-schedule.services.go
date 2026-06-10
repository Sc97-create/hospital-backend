package admins

import (
	"errors"
	"hospital-backend/internal/admins/dto"
	"time"

	"github.com/google/uuid"
)

type OrganisationScheduleService struct {
	repo OrganisationScheduleRepository
}

func NewOrganisationScheduleService(repos OrganisationScheduleRepository) *OrganisationScheduleService {
	return &OrganisationScheduleService{repo: repos}
}

func (s *OrganisationScheduleService) Create(reqModel dto.OrgScheduleReq) error {
	orgSchedModel := s.toOrgSchedModel(reqModel)
	err := s.repo.Create(&orgSchedModel)
	if err != nil {
		return errors.New("failed to insert record")
	}
	return nil
}

func (s *OrganisationScheduleService) GetScheduleByOrganisationID(organisationID string) (dto.GetResponse, error) {
	query := `select id,start_time,end_time,slot_duration,break_start_time,break_end_time from organisation_schedules where organisation_id=$1`
	OrganisationSchedule, err := s.repo.GetByOrganisationID(query, organisationID)
	if err != nil {
		return dto.GetResponse{}, nil
	}
	response := s.toResponseModel(OrganisationSchedule)

	// Implement logic to retrieve schedule by organisation ID
	return response, nil
}
func (s *OrganisationScheduleService) toResponseModel(organisationSchedule OrganisationSchedule) dto.GetResponse {
	starttime, _ := time.Parse("15:04", organisationSchedule.StartTime)
	endtime, _ := time.Parse("15:04", organisationSchedule.EndTime)
	breakStarttime, _ := time.Parse("15:04", organisationSchedule.BreakStartTime)
	breakEndtime, _ := time.Parse("15:04", organisationSchedule.BreakEndTime)
	return dto.GetResponse{
		ID:             organisationSchedule.ID,
		Starttime:      starttime,
		Endtime:        endtime,
		BreakStarttime: breakStarttime,
		BreakEndtime:   breakEndtime,
		Slotduration:   organisationSchedule.SlotDuration,
	}
}
func (s *OrganisationScheduleService) toOrgSchedModel(reqModel dto.OrgScheduleReq) OrganisationSchedule {
	return OrganisationSchedule{
		ID:             uuid.NewString(),
		OrganisationID: reqModel.OrganisationID,
		DayOfWeek:      reqModel.WeekDays,
		StartTime:      reqModel.StartTime,
		EndTime:        reqModel.EndTime,
		BreakStartTime: reqModel.BreakStartTime,
		BreakEndTime:   reqModel.BreakEndTime,
		SlotDuration:   int(reqModel.SlotDuration),
		IsClosed:       reqModel.IsClosed,
		CreatedAt:      time.Now(),
	}
}
