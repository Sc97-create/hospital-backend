package employee

import (
	"errors"
	"hospital-backend/internal/employee/dto"
	"hospital-backend/internal/employee/utils"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type EmployeeControllers interface {
	Add(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
	FindByID(c *fiber.Ctx) error
	FindMany(c *fiber.Ctx) error
	CreateAdmin(c *fiber.Ctx) error
	UpdateUser(c *fiber.Ctx) error
	FindDoctors(c *fiber.Ctx) error
}
type EmployeeServicer interface {
	CreateEmployee(log *zap.Logger, payload dto.EmpRequest) (id string, err error)
	DeleteEmployee(log *zap.Logger, userID string) (err error)
	FindOne(log *zap.Logger, id string) (dto.EmployeeResponse, error)
	FindMany(log *zap.Logger, req dto.FindManyRequest) (employeeResp []dto.EmployeeResponse, total int64, err error)
	CreateAdminProf(log *zap.Logger, payload dto.EmpRequest) (userID string, err error)
	UpdateAdminProf(log *zap.Logger, payload dto.UpdateRequest) (err error)
	FindDoctors(log *zap.Logger, search string, organisationID string) (u []dto.Doctor, err error)
}

type EmployeeController struct {
	EmployeeService EmployeeServicer
}

func NewEmployeeControllerInterface(employeeService EmployeeServicer) *EmployeeController {
	return &EmployeeController{EmployeeService: employeeService}
}

func (e *EmployeeController) Add(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	payload, err := params.New(c)
	if err != nil {
		logger.Warn("employee create request invalid", zap.Error(err))
		return wrapError.Wrap(err, c, 409)
	}
	userReq, field, err := e.toEmpRequest(payload)
	if err != nil {
		logger.Warn("employee create request invalid", zap.String("field", field))
		return wrapError.Wrap(err, c, 409)
	}
	logger.Info("employee create attempt",
		zap.String("organisation_id", userReq.OrganisationID),
		zap.String("role_id", userReq.RoleID),
		zap.String("department_id", userReq.DepartmentID),
	)
	_, err = e.EmployeeService.CreateEmployee(logger, userReq)
	if err != nil {
		return wrapError.Wrap(safeEmployeeError(err, wrapError.ErrEmployeeCreateFailed), c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"message": "success"})
}

func (e *EmployeeController) toEmpRequest(payload *params.Payload) (dto.EmpRequest, string, error) {
	var req dto.EmpRequest
	if field, err := e.parseEmpIdentity(payload, &req); err != nil {
		return dto.EmpRequest{}, field, err
	}
	if field, err := e.validateEmpIdentity(&req); err != nil {
		return dto.EmpRequest{}, field, err
	}
	if field, err := e.parseEmpJobDetails(payload, &req); err != nil {
		return dto.EmpRequest{}, field, err
	}
	if field, err := e.parseEmpShiftAndEmergency(payload, &req); err != nil {
		return dto.EmpRequest{}, field, err
	}
	return req, "", nil
}

func (e *EmployeeController) parseEmpIdentity(payload *params.Payload, req *dto.EmpRequest) (string, error) {
	var err error
	req.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return "organisation_id", err
	}
	req.FirstName, err = payload.Getstring("first_name")
	if err != nil {
		return "first_name", err
	}
	req.LastName, err = payload.Getstring("last_name")
	if err != nil {
		return "last_name", err
	}
	req.MobileNumber, err = payload.Getstring("mobile_number")
	if err != nil {
		return "mobile_number", err
	}
	req.PhoneNumber = req.MobileNumber
	req.EmailID, err = payload.Getstring("email_id")
	if err != nil {
		return "email_id", err
	}
	req.Address, err = payload.Getstring("address")
	if err != nil {
		return "address", err
	}
	return "", nil
}

func (e *EmployeeController) validateEmpIdentity(req *dto.EmpRequest) (string, error) {
	req.FirstName = strings.TrimSpace(req.FirstName)
	if !utils.IsValidName(req.FirstName) {
		return "first_name", wrapError.ErrInvalidRequest
	}
	req.LastName = strings.TrimSpace(req.LastName)
	if !utils.IsValidName(req.LastName) {
		return "last_name", wrapError.ErrInvalidRequest
	}
	mobile, ok := utils.NormalizeMobile(req.MobileNumber)
	if !ok {
		return "mobile_number", wrapError.ErrInvalidRequest
	}
	req.MobileNumber = mobile
	req.PhoneNumber = mobile
	req.EmailID = strings.TrimSpace(req.EmailID)
	if !utils.IsValidEmail(req.EmailID) {
		return "email_id", wrapError.ErrInvalidRequest
	}
	return "", nil
}

func (e *EmployeeController) parseEmpJobDetails(payload *params.Payload, req *dto.EmpRequest) (string, error) {
	var err error
	req.DateOfBirth, err = payload.Getstring("date_of_birth")
	if err != nil {
		return "date_of_birth", err
	}
	req.DateOfJoining, err = payload.Getstring("date_of_joining")
	if err != nil {
		return "date_of_joining", err
	}
	req.RoleID, err = payload.Getstring("role_id")
	if err != nil {
		return "role_id", err
	}
	req.DepartmentID, err = payload.Getstring("dept_id")
	if err != nil {
		return "dept_id", err
	}
	req.LicenseNo, err = payload.Getstring("license_no")
	if err != nil {
		return "license_no", err
	}
	req.Qualification, err = payload.Getstring("qualification")
	if err != nil {
		return "qualification", err
	}
	req.EmployeeType, err = payload.Getstring("employee_type")
	if err != nil {
		return "employee_type", err
	}
	return "", nil
}

func (e *EmployeeController) parseEmpShiftAndEmergency(payload *params.Payload, req *dto.EmpRequest) (string, error) {
	if field, err := e.parseShiftTimings(payload, req); err != nil {
		return field, err
	}
	return e.parseEmergencyDetails(payload, req)
}

func (e *EmployeeController) parseShiftTimings(payload *params.Payload, req *dto.EmpRequest) (string, error) {
	shift, err := payload.GetObject("shift_timings")
	if err != nil {
		return "shift_timings", err
	}
	req.ShiftTimings.StartTime, err = shift.Getstring("start_time")
	if err != nil {
		return "shift_timings.start_time", err
	}
	req.ShiftTimings.EndTime, err = shift.Getstring("end_time")
	if err != nil {
		return "shift_timings.end_time", err
	}
	return "", nil
}

func (e *EmployeeController) parseEmergencyDetails(payload *params.Payload, req *dto.EmpRequest) (string, error) {
	emergency, err := payload.GetObject("emergency_details")
	if err != nil {
		return "emergency_details", err
	}
	req.EmergencyDetails.Email, err = emergency.Getstring("email")
	if err != nil {
		return "emergency_details.email", err
	}
	req.EmergencyDetails.Name, err = emergency.Getstring("name")
	if err != nil {
		return "emergency_details.name", err
	}
	req.EmergencyDetails.Contact, err = emergency.Getstring("contact")
	if err != nil {
		return "emergency_details.contact", err
	}
	return "", nil
}
func (e *EmployeeController) Delete(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	payload, err := params.New(c)
	if err != nil {
		logger.Warn("employee delete request invalid", zap.Error(err))
		return wrapError.Wrap(err, c, 409)
	}
	userID, err := payload.Getstring("user_id")
	if err != nil {
		logger.Warn("employee delete request invalid", zap.String("field", "user_id"))
		return wrapError.Wrap(err, c, 409)
	}
	logger.Info("employee delete attempt", zap.String("user_id", userID))

	err = e.EmployeeService.DeleteEmployee(logger, userID)
	if err != nil {
		return wrapError.Wrap(safeEmployeeError(err, wrapError.ErrEmployeeDeleteFailed), c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"message": "deleted successfully", "code": "xyz123"})
}
func (e *EmployeeController) FindByID(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	userID := c.Query("user_id")
	if strings.TrimSpace(userID) == "" {
		logger.Warn("employee get request invalid", zap.String("reason", "missing_user_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, 409)
	}
	logger.Info("employee get attempt", zap.String("user_id", userID))

	user, err := e.EmployeeService.FindOne(logger, userID)
	if err != nil {
		return wrapError.Wrap(safeEmployeeError(err, wrapError.ErrEmployeeFetchFailed), c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"data": user, "message": "user fetched successfully"})
}
func (e *EmployeeController) FindMany(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	payload := dto.FindManyRequest{}
	if err = c.QueryParser(&payload); err != nil {
		logger.Warn("employee list request invalid", zap.Error(err))
		return wrapError.Wrap(err, c, 409)
	}
	logger.Info("employee list attempt", zap.String("organisation_id", payload.OrganisationID))
	employees, total, err := e.EmployeeService.FindMany(logger, payload)
	if err != nil {
		return wrapError.Wrap(safeEmployeeError(err, wrapError.ErrEmployeesFetchFailed), c, 409)
	}
	var response dto.EmployeeListResponse
	response.Data = employees
	response.Total = total
	response.TotalCount = total
	response.Code = 200
	err = c.Status(200).JSON(&response)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return
}
func (e *EmployeeController) CreateAdmin(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	payload, err := params.New(c)
	if err != nil {
		logger.Warn("employee admin create request invalid", zap.Error(err))
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
	AdminReq.EmailID, err = payload.Getstring("email_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	AdminReq.PhoneNumber, err = payload.Getstring("mob_no")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	logger.Info("employee admin create attempt", zap.String("organisation_id", AdminReq.OrganisationID))
	userID, err := e.EmployeeService.CreateAdminProf(logger, AdminReq)
	if err != nil {
		return wrapError.Wrap(safeEmployeeError(err, wrapError.ErrEmployeeCreateFailed), c, 409)
	}
	return c.JSON(fiber.Map{"message": "account created successfully", "code": 200, "user_id": userID})
}
func (e *EmployeeController) UpdateUser(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	payload, err := params.New(c)
	if err != nil {
		logger.Warn("employee update request invalid", zap.Error(err))
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
		logger.Warn("employee update request invalid", zap.String("reason", "password_mismatch"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, 409)
	}
	logger.Info("employee update attempt", zap.String("user_id", AdminReq.UserID))
	err = e.EmployeeService.UpdateAdminProf(logger, AdminReq)
	if err != nil {
		return wrapError.Wrap(safeEmployeeError(err, wrapError.ErrEmployeeUpdateFailed), c, 409)
	}
	return c.JSON(fiber.Map{"message": "updated successfully", "code": 200})
}

func (e *EmployeeController) FindDoctors(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	name := c.Query("name")
	organisationID := c.Query("organisation_id")
	if strings.TrimSpace(organisationID) == "" {
		logger.Warn("doctor list request invalid", zap.String("reason", "missing_organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, 409)
	}
	logger.Info("doctor list attempt", zap.String("organisation_id", organisationID))

	users, err := e.EmployeeService.FindDoctors(logger, name, organisationID)
	if err != nil {
		return wrapError.Wrap(safeEmployeeError(err, wrapError.ErrDoctorsFetchFailed), c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"data": users, "message": "doctors fetched successfully", "code": 200})
}

func safeEmployeeError(err, fallback error) error {
	switch {
	case errors.Is(err, ErrEmployeeNotFound),
		errors.Is(err, wrapError.ErrInvalidRequest),
		errors.Is(err, wrapError.ErrEmployeeCreateFailed),
		errors.Is(err, wrapError.ErrEmployeeFetchFailed),
		errors.Is(err, wrapError.ErrEmployeesFetchFailed),
		errors.Is(err, wrapError.ErrEmployeeUpdateFailed),
		errors.Is(err, wrapError.ErrEmployeeDeleteFailed),
		errors.Is(err, wrapError.ErrDoctorsFetchFailed):
		return err
	default:
		return fallback
	}
}
