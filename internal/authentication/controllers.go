package authentication

import (
	"errors"
	"hospital-backend/internal/authentication/dto"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

type IAuthService interface {
	Login(c *fiber.Ctx) (dto.LoginResponse, error)
	Refresh(c *fiber.Ctx) error
}
type AuthController struct {
	AuthService *UserService
}

func NewAuthController(service *UserService) *AuthController {
	return &AuthController{AuthService: service}
}
func (a *AuthController) Login(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	loginRequest := dto.LoginUser{}
	loginRequest.Username, err = payload.Getstring("user_name")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	loginRequest.Password, err = payload.Getstring("password")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}

	userResp, err := a.AuthService.Login(loginRequest)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	a.setRefreshToken(c, userResp.RefreshToken)
	var response dto.LoginResponse
	response.UserID = userResp.UserID
	response.Token = userResp.Token
	response.RefreshToken = userResp.RefreshToken
	response.OrganisationID = userResp.OrganisationID
	response.Message = "user logged in successfully"
	if err := c.Status(200).JSON(response); err != nil {
		return err
	}
	return
}
func (a *AuthController) Refresh(c *fiber.Ctx) (err error) {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return wrapError.Wrap(errors.New("refresh token not found"), c, 409)
	}
	tokenresp, err := a.AuthService.RefreshToken(refreshToken)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	a.setRefreshToken(c, tokenresp.RefreshToken)
	response := make(map[string]any)
	response["accesstoken"] = tokenresp.Token
	response["message"] = "user logged in successfully"
	if err := c.Status(200).JSON(response); err != nil {
		return err
	}
	return
}
func (a *AuthController) setRefreshToken(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:  "refresh_token",
		Value: token,
		//HTTPOnly: true,
		//Secure:   false,
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/",
		MaxAge:   int(1440 * 60),
	})

}
