package employee

import (
	"hospital-backend/internal/employee/dto"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

type EmployeeControllers interface {
	AddHandler(c *fiber.Ctx) error
	DeleteHandler(c *fiber.Ctx) error
	FindByIDHandler(c *fiber.Ctx) error
	FindManyHandler(c *fiber.Ctx) error
	CreateAdmin(c *fiber.Ctx) error
	UpdateUser(c *fiber.Ctx) error
	FindDoctorsHandler(c *fiber.Ctx) error
}
type EmployeeController struct {
	EmployeeService *EmployeeService
}

func NewEmployeeControllerInterface(employeeService *EmployeeService) *EmployeeController {
	return &EmployeeController{EmployeeService: employeeService}
}

func (e *EmployeeController) AddHandler(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	UserReq := dto.EmpRequest{}
	UserReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	UserReq.UserName, err = payload.Getstring("username")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	UserReq.EmailID, err = payload.Getstring("email_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	UserReq.RoleID, err = payload.Getstring("role_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	UserReq.DepartmentID, err = payload.Getstring("department_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	UserReq.PhoneNumber, err = payload.Getstring("phone_number")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	_, err = e.EmployeeService.CreateEmployee(UserReq)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"message": "success"})
}
func (e *EmployeeController) DeleteHandler(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	userID, err := payload.Getstring("user_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}

	err = e.EmployeeService.DeleteEmployee(userID)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"message": "deleted successfully", "code": "xyz123"})
}
func (e *EmployeeController) FindByIDHandler(c *fiber.Ctx) (err error) {
	userID := c.Query("user_id")

	user, err := e.EmployeeService.FindOne(userID)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"data": user, "message": "user fetched successfully"})
}
func (e *EmployeeController) FindManyHandler(c *fiber.Ctx) (err error) {
	param, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	limit, err := param.Getint("limit")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	pageno, err := param.Getint("page_no")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}

	users, err := e.EmployeeService.FindMany(limit, pageno)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	resp := make(map[string]any)
	resp["data"] = users
	resp["code"] = 200
	err = c.JSON(&resp)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return
}
func (e *EmployeeController) CreateAdmin(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq := dto.EmpRequest{}
	AdminReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq.FirstName, err = payload.Getstring("first_name")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq.LastName, err = payload.Getstring("last_name")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq.Password, err = payload.Getstring("password")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq.ConfirmPassword, err = payload.Getstring("confirm_password")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq.AssignDefRoles, _ = payload.GetBool("assign_default_roles")
	AdminReq.AssignDefDept, _ = payload.GetBool("assign_default_departments")
	AdminReq.AssignDefRolePerm, _ = payload.GetBool("assign_default_role_permissions")
	AdminReq.EmailID, err = payload.Getstring("email_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq.PhoneNumber, err = payload.Getstring("mob_no")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	userID, err := e.EmployeeService.CreateAdminProf(AdminReq)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.JSON(fiber.Map{"message": "account created successfully", "code": 200, "user_id": userID})
}
func (e *EmployeeController) UpdateUser(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq := dto.UpdateRequest{}
	AdminReq.UserID, _ = payload.Getstring("user_id")
	AdminReq.MobileNumber, _ = payload.Getstring("mob_no")
	AdminReq.FirstName, _ = payload.Getstring("first_name")
	AdminReq.LastName, _ = payload.Getstring("last_name")
	AdminReq.Password, _ = payload.Getstring("password")
	confirmPassword, _ := payload.Getstring("confirm_password")
	if AdminReq.Password != confirmPassword {
		return wrapError.Wrap(err, c, 409)
	}
	err = e.EmployeeService.UpdateAdminProf(AdminReq)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.JSON(fiber.Map{"message": "updated successfully", "code": 200})
}

func (e *EmployeeController) FindDoctorsHandler(c *fiber.Ctx) (err error) {
	name := c.Query("name")

	users, err := e.EmployeeService.FindDoctors(name)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"data": users, "message": "doctors fetched successfully", "code": 200})
}
