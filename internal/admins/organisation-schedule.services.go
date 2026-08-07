package admins

import (
	"errors"
	"hospital-backend/internal/admins/dto"
	wrapError "hospital-backend/shared/error"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrganisationScheduleService struct {
	repo OrganisationScheduleRepository
}

func NewOrganisationScheduleService(repos OrganisationScheduleRepository) *OrganisationScheduleService {
	return &OrganisationScheduleService{repo: repos}
}

func (s *OrganisationScheduleService) Create(log *zap.Logger, reqModel dto.OrgScheduleReq) error {
	log = ensureLog(log)
	orgSchedModel := s.toOrgSchedModel(reqModel)
	err := s.repo.Create(log, &orgSchedModel)
	if err != nil {
		log.Error("organisation schedule create failed",
			zap.String("organisation_id", reqModel.OrganisationID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return wrapError.ErrOrgScheduleCreateFailed
	}
	log.Info("organisation schedule create success",
		zap.String("organisation_id", reqModel.OrganisationID),
		zap.String("schedule_id", orgSchedModel.ID),
		zap.Int("slot_duration", orgSchedModel.SlotDuration),
		zap.Int("week_off_count", len(reqModel.WeekDays)),
		zap.Bool("is_closed", reqModel.IsClosed),
	)
	return nil
}

func (s *OrganisationScheduleService) GetScheduleByOrganisationID(log *zap.Logger, organisationID string) (dto.GetResponse, error) {
	log = ensureLog(log)
	organisationID = strings.TrimSpace(organisationID)
	if organisationID == "" {
		return dto.GetResponse{}, wrapError.ErrInvalidRequest
	}

	log.Debug("organisation schedule get by org", zap.String("organisation_id", organisationID))

	query := `select id,start_time,end_time,slot_duration,break_start_time,break_end_time from organisation_schedules where organisation_id=$1`
	OrganisationSchedule, err := s.repo.GetByOrganisationID(log, query, organisationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("organisation schedule get failed",
				zap.String("organisation_id", organisationID),
				zap.String("reason", "not_found"),
			)
			return dto.GetResponse{}, wrapError.ErrOrgScheduleNotFound
		}
		log.Error("organisation schedule get failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.GetResponse{}, wrapError.ErrOrgScheduleFetchFailed
	}

	response := s.toResponseModel(OrganisationSchedule)
	log.Debug("organisation schedule get success",
		zap.String("organisation_id", organisationID),
		zap.String("schedule_id", response.ID),
		zap.Int("slot_duration", response.Slotduration),
	)
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
