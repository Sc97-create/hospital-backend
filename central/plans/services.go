package plans

import (
	dto "hospital-backend/central/plans/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

type PlanService struct {
	Repo PlanRepository
}

func NewPlanService(repo PlanRepository) *PlanService {
	return &PlanService{Repo: repo}
}

func (s *PlanService) ListPlans(log *zap.Logger) ([]dto.PlanDetail, error) {
	log = ensureLog(log)
	rows, err := s.Repo.ListActive(log)
	if err != nil {
		log.Error("plan list failed", zap.String("reason", "db_list"), zap.Error(err))
		return nil, wrapError.ErrPlanFetchFailed
	}

	out := make([]dto.PlanDetail, 0, len(rows))
	for _, plan := range rows {
		out = append(out, toPlanDetail(plan))
	}
	log.Info("plan list success", zap.Int("count", len(out)))
	return out, nil
}

func toPlanDetail(plan Plan) dto.PlanDetail {
	details := map[string]any(plan.PlanDetails)
	if details == nil {
		details = map[string]any{}
	}
	status := plan.Status
	if status == "" {
		status = constants.StatusActive
	}
	return dto.PlanDetail{
		ID:                    plan.ID,
		Name:                  plan.Name,
		Status:                status,
		MonthlyPrice:          plan.MonthlyPrice,
		Discount:              plan.Discount,
		EffectiveMonthlyPrice: plan.MonthlyPrice * (1 - plan.Discount/100),
		PlanDetails:           details,
	}
}
