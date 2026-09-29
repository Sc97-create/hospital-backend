package plans

import (
	"errors"

	dto "hospital-backend/central/plans/dto"
	"hospital-backend/central/middleware"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type PlanServicer interface {
	ListPlans(log *zap.Logger) ([]dto.PlanDetail, error)
}

type PlanController struct {
	Service PlanServicer
}

type IPlanController interface {
	List(c *fiber.Ctx) error
}

func NewIPlanController(service PlanServicer) IPlanController {
	return &PlanController{Service: service}
}

func (pc *PlanController) List(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	logger.Info("plan list attempt")

	plans, err := pc.Service.ListPlans(logger)
	if err != nil {
		return wrapListError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "plans fetched successfully",
		"plans":   plans,
	})
}

func wrapListError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrPlanFetchFailed):
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	default:
		return wrapError.Wrap(wrapError.ErrPlanFetchFailed, c, fiber.StatusInternalServerError)
	}
}
