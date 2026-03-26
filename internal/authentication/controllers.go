package authentication

import (
	"hospital-backend/internal/authentication/dto"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

func Login(c *fiber.Ctx, service *UserService) (err error) {
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

	userResp, err := service.Login(loginRequest)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	if userResp.RefreshToken != "" {
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    userResp.RefreshToken,
			HTTPOnly: true,
			//Secure:   true,
			SameSite: "Strict",
			Path:     "/",
		})
	}

	response := make(map[string]any)
	response["user_id"] = userResp.UserID
	response["accesstoken"] = userResp.Token
	response["message"] = "user logged in successfully"
	if err := c.Status(200).JSON(response); err != nil {
		return err
	}
	return
}
