package jwt

import (
	"strings"

	"hospital-backend/pkg/logger"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// AccessTokenIssuer is the JWT surface used by the internal access-token API.
type AccessTokenIssuer interface {
	AccessToken(userID, organisationID string) (string, error)
}

type accessTokenRequest struct {
	UserID         string `json:"user_id"`
	OrganisationID string `json:"organisation_id"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type Controller struct {
	Issuer AccessTokenIssuer
}

func NewController(issuer AccessTokenIssuer) *Controller {
	return &Controller{Issuer: issuer}
}

// CreateAccessToken is the internal API central calls to mint an access token.
func (c *Controller) CreateAccessToken(ctx *fiber.Ctx) error {
	log := requestLogger(ctx)
	var req accessTokenRequest
	if err := ctx.BodyParser(&req); err != nil {
		log.Warn("access token request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	if req.UserID == "" {
		log.Warn("access token request invalid", zap.String("reason", "missing_user_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, ctx, fiber.StatusBadRequest)
	}
	if c.Issuer == nil {
		log.Error("access token create failed", zap.String("reason", "issuer_missing"))
		return wrapError.Wrap(wrapError.ErrAccessTokenCreateFailed, ctx, fiber.StatusInternalServerError)
	}

	log.Info("access token create attempt", zap.String("user_id", req.UserID))
	token, err := c.Issuer.AccessToken(req.UserID, req.OrganisationID)
	if err != nil {
		log.Error("access token create failed",
			zap.String("user_id", req.UserID),
			zap.String("reason", "issue"),
			zap.Error(err),
		)
		return wrapError.Wrap(wrapError.ErrAccessTokenCreateFailed, ctx, fiber.StatusInternalServerError)
	}
	log.Info("access token create success", zap.String("user_id", req.UserID))
	return ctx.Status(fiber.StatusOK).JSON(accessTokenResponse{AccessToken: token})
}

func requestLogger(ctx *fiber.Ctx) *zap.Logger {
	if l, ok := ctx.Locals("req_logger").(*zap.Logger); ok {
		return l
	}
	if logger.Log != nil {
		return logger.Log
	}
	return zap.NewNop()
}
