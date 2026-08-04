package prescription

import (
	"errors"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/pkg/middleware"
	"hospital-backend/shared/params"

	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type PrescriptionController struct {
	PService     *PrescriptionService
	PItemService *PrescriptionItemServ
}

func NewPrescriptionController(PService *PrescriptionService, PItems *PrescriptionItemServ) *PrescriptionController {
	return &PrescriptionController{PService: PService, PItemService: PItems}
}

type IPrescriptionController interface {
	CreatePrescription(c *fiber.Ctx) error
	GetPrescriptionsByPatientID(c *fiber.Ctx) error
	AddPrescriptionItems(c *fiber.Ctx) error
	UpdatePrescriptionItem(c *fiber.Ctx) error
	FindMany(c *fiber.Ctx) error
	FindByStatus(c *fiber.Ctx) error
	FindPrescriptionByID(c *fiber.Ctx) error
	UpdateStatus(c *fiber.Ctx) error
	FindMedicineDetInfo(c *fiber.Ctx) error
}

func (PresC *PrescriptionController) CreatePrescription(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("prescription create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	requestmap := dto.CreatePrescriptionRequest{}
	requestmap.AppointmentID, err = payload.Getstring("appointment_id")
	if err != nil {
		logger.Warn("prescription create request invalid", zap.Error(err), zap.String("field", "appointment_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestmap.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		logger.Warn("prescription create request invalid", zap.Error(err), zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestmap.PrescribedBy, err = payload.Getstring("prescribed_by")
	if err != nil {
		logger.Warn("prescription create request invalid", zap.Error(err), zap.String("field", "prescribed_by"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	medicines, err := payload.GetChildren("medicine_array")
	if err != nil {
		logger.Warn("prescription create request invalid", zap.Error(err), zap.String("field", "medicine_array"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestmap.MedicineArray = PresC.toMedicineArray(medicines)

	logger.Info("prescription create attempt",
		zap.String("organisation_id", requestmap.OrganisationID),
		zap.String("appointment_id", requestmap.AppointmentID),
		zap.String("prescribed_by", requestmap.PrescribedBy),
		zap.Int("item_count", len(requestmap.MedicineArray)),
	)

	id, err := PresC.PService.CreatePrescription(logger, requestmap)
	if err != nil {
		return PresC.wrapCreateError(c, err)
	}

	var response dto.CreatePrescriptionResponse
	response.Code = "200"
	response.Data = dto.Data{ID: id}
	response.Message = "prescription saved successfully"
	return c.Status(fiber.StatusOK).JSON(response)
}

func (PresC *PrescriptionController) wrapCreateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrAppointmentNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrMedicineAlreadyPresent):
		return wrapError.Wrap(err, c, fiber.StatusConflict)
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	default:
		var ve *validationError
		if errors.As(err, &ve) {
			return wrapError.Wrap(err, c, fiber.StatusBadRequest)
		}
		return wrapError.Wrap(wrapError.ErrPrescriptionCreateFailed, c, fiber.StatusInternalServerError)
	}
}

func (PresC *PrescriptionController) FindMany(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	var requestmap dto.FindManyRequest
	err := c.QueryParser(&requestmap)
	if err != nil {
		logger.Warn("prescription list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if requestmap.OrganisationID == "" {
		logger.Warn("prescription list request invalid", zap.String("reason", "missing_organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("prescription list attempt",
		zap.String("organisation_id", requestmap.OrganisationID),
		zap.Int("limit", requestmap.Limit),
		zap.Int("offset", requestmap.Offset),
		zap.Bool("has_search", requestmap.Search != ""),
	)

	prescriptions, totalcount, err := PresC.PService.FindMany(logger, requestmap.Limit, requestmap.Offset, requestmap.OrganisationID, requestmap.Search)
	if err != nil {
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(PresC.toPrescriptionListResponse(prescriptions, totalcount))
}

func (PresC *PrescriptionController) FindByStatus(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	var requestmap dto.FindByStatusRequest
	err := c.QueryParser(&requestmap)
	if err != nil {
		logger.Warn("prescription status list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if requestmap.OrganisationID == "" {
		logger.Warn("prescription status list request invalid", zap.String("reason", "missing_organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if requestmap.Status == "" {
		logger.Warn("prescription status list request invalid", zap.String("reason", "missing_status"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("prescription status list attempt",
		zap.String("organisation_id", requestmap.OrganisationID),
		zap.String("status", requestmap.Status),
		zap.Int("limit", requestmap.Limit),
		zap.Int("offset", requestmap.Offset),
	)

	prescriptions, totalcount, err := PresC.PService.FindByStatus(logger, requestmap.Limit, requestmap.Offset, requestmap.OrganisationID, requestmap.Status)
	if err != nil {
		if errors.Is(err, wrapError.ErrInvalidRequest) {
			return wrapError.Wrap(err, c, fiber.StatusBadRequest)
		}
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(PresC.toPrescriptionListResponse(prescriptions, totalcount))
}

func (PresC *PrescriptionController) toPrescriptionListResponse(prescriptions []dto.PrescriptionListItem, totalCount int64) dto.FindManyResponse {
	if prescriptions == nil {
		prescriptions = []dto.PrescriptionListItem{}
	}
	return dto.FindManyResponse{
		Code:       "200",
		Message:    "prescriptions fetched successfully",
		Data:       prescriptions,
		TotalCount: totalCount,
	}
}

func (Presc *PrescriptionController) AddPrescriptionItems(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("prescription items add request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var requestMap dto.UpdateRequest
	requestMap.PrescriptionID, err = payload.Getstring("prescription_id")
	if err != nil {
		logger.Warn("prescription items add request invalid", zap.Error(err), zap.String("field", "prescription_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestMap.UserID, err = payload.Getstring("prescribed_by")
	if err != nil {
		logger.Warn("prescription items add request invalid", zap.Error(err), zap.String("field", "prescribed_by"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	medicineArr, err := payload.GetChildren("medicine_array")
	if err != nil {
		logger.Warn("prescription items add request invalid", zap.Error(err), zap.String("field", "medicine_array"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestMap.MedicineArr = Presc.toMedicineArray(medicineArr)

	logger.Info("prescription items add attempt",
		zap.String("prescription_id", requestMap.PrescriptionID),
		zap.String("prescribed_by", requestMap.UserID),
		zap.Int("item_count", len(requestMap.MedicineArr)),
	)

	err = Presc.PService.AddPrescriptionItems(logger, requestMap)
	if err != nil {
		if errors.Is(err, wrapError.ErrMedicineAlreadyPresent) {
			return wrapError.Wrap(err, c, fiber.StatusConflict)
		}
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	}

	var response dto.CreatePrescriptionResponse
	response.Code = "200"
	response.Message = "prescription updated successfully"
	response.Data = dto.Data{ID: requestMap.PrescriptionID}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (Presc *PrescriptionController) UpdatePrescriptionItem(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("prescription item update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var requestMap dto.UpdatePrescriptionItemRequest
	requestMap.PrescriptionItemID, err = payload.Getstring("prescription_item_id")
	if err != nil {
		logger.Warn("prescription item update request invalid", zap.Error(err), zap.String("field", "prescription_item_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestMap.MedicineID, err = payload.Getstring("medicine_id")
	if err != nil {
		logger.Warn("prescription item update request invalid", zap.Error(err), zap.String("field", "medicine_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	requestMap.DurationDay, _ = payload.Getfloat("duration")
	requestMap.DurationType, _ = payload.Getstring("duration_type")
	requestMap.FoodInstruction, _ = payload.Getstring("food_instruction")
	requestMap.Morning, _ = payload.Getfloat("morning")
	requestMap.Afternoon, _ = payload.Getfloat("afternoon")
	requestMap.Night, _ = payload.Getfloat("night")

	logger.Info("prescription item update attempt",
		zap.String("prescription_item_id", requestMap.PrescriptionItemID),
		zap.String("medicine_id", requestMap.MedicineID),
	)

	err = Presc.PItemService.UpdatePrescriptionItemByID(logger, requestMap)
	if err != nil {
		switch {
		case errors.Is(err, wrapError.ErrPrescriptionItemNotFound):
			return wrapError.Wrap(err, c, fiber.StatusNotFound)
		case errors.Is(err, wrapError.ErrCannotEditDispensedItem):
			return wrapError.Wrap(err, c, fiber.StatusConflict)
		default:
			return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
		}
	}

	var response dto.CreatePrescriptionResponse
	response.Code = "200"
	response.Message = "prescription item updated successfully"
	response.Data = dto.Data{ID: requestMap.PrescriptionItemID}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (PresC *PrescriptionController) FindPrescriptionByID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	prescriptionID := c.Query("prescription_id")
	limit := c.QueryFloat("limit", 10)
	offset := c.QueryFloat("offset", 0)

	if prescriptionID == "" {
		logger.Warn("prescription items get request invalid", zap.String("reason", "missing_prescription_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("prescription items get attempt",
		zap.String("prescription_id", prescriptionID),
		zap.Float64("limit", limit),
		zap.Float64("offset", offset),
	)

	medicines, totalCount, err := PresC.PItemService.GetPrescriptionsByPIDWithLimit(logger, prescriptionID, limit, offset)
	if err != nil {
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	}

	var response dto.FindPrescriptionByIDResponse
	response.Code = "200"
	response.Message = "prescription fetched successfully"
	response.Data.MedicineResponse = medicines
	response.Data.TotalCount = int(totalCount)
	return c.Status(fiber.StatusOK).JSON(response)
}

func (PresC *PrescriptionController) UpdateStatus(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("prescription status update request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	prescriptionID, err := payload.Getstring("prescription_id")
	if err != nil {
		logger.Warn("prescription status update request invalid", zap.Error(err), zap.String("field", "prescription_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	appointmentID, _ := payload.Getstring("appointment_id")
	status, err := payload.Getstring("status")
	if err != nil {
		logger.Warn("prescription status update request invalid", zap.Error(err), zap.String("field", "status"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("prescription status update attempt",
		zap.String("prescription_id", prescriptionID),
		zap.String("status", status),
		zap.Bool("has_appointment_id", appointmentID != ""),
	)

	err = PresC.PService.UpdateManualStatus(logger, prescriptionID, appointmentID, status)
	if err != nil {
		if errors.Is(err, wrapError.ErrInvalidRequest) {
			return wrapError.Wrap(err, c, fiber.StatusBadRequest)
		}
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	}

	var response dto.CreatePrescriptionResponse
	response.Code = "200"
	response.Message = "prescription status updated successfully"
	response.Data = dto.Data{ID: prescriptionID}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (PresC *PrescriptionController) GetPrescriptionByAppointmentID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("prescription appointment list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var reqmodel dto.PresPatients
	reqmodel.AppointmentID, err = payload.Getstring("appointment_id")
	if err != nil {
		logger.Warn("prescription appointment list request invalid", zap.Error(err), zap.String("field", "appointment_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqmodel.Pageno, err = payload.Getfloat("page_no")
	if err != nil {
		logger.Warn("prescription appointment list request invalid", zap.Error(err), zap.String("field", "page_no"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqmodel.Limit, err = payload.Getfloat("limit")
	if err != nil {
		logger.Warn("prescription appointment list request invalid", zap.Error(err), zap.String("field", "limit"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqmodel.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		logger.Warn("prescription appointment list request invalid", zap.Error(err), zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("prescription appointment list attempt",
		zap.String("appointment_id", reqmodel.AppointmentID),
		zap.String("organisation_id", reqmodel.OrganisationID),
		zap.Float64("limit", reqmodel.Limit),
		zap.Float64("page_no", reqmodel.Pageno),
	)

	response, err := PresC.PService.GetPrescriptionByAppointmentID(logger, reqmodel)
	if err != nil {
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (PresC *PrescriptionController) GetPrescriptionsByPatientID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	var request dto.PatientPrescriptionsRequest
	if err := c.QueryParser(&request); err != nil {
		logger.Warn("prescription patient list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if request.PatientID == "" {
		logger.Warn("prescription patient list request invalid", zap.String("reason", "missing_patient_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("prescription patient list attempt",
		zap.String("patient_id", request.PatientID),
		zap.Int("limit", request.Limit),
		zap.Int("page_no", request.PageNo),
	)

	response, err := PresC.PService.GetPrescriptionsByPatientID(logger, request)
	if err != nil {
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func (Presc *PrescriptionController) validateIDs(args ...any) error {
	if len(args) == 0 {
		return nil
	}
	for _, each := range args {
		switch each.(type) {
		case string:
			if each == "" {
				return errors.New(" please pass the required query value")
			}
		}
	}
	return nil
}

func (Presc *PrescriptionController) toMedicineArray(medicine []*params.Payload) []dto.MedicineArray {
	var medicineArray []dto.MedicineArray
	for _, each := range medicine {
		MedicineID, _ := each.Getstring("medicine_id")
		DurationDay, _ := each.Getfloat("duration")
		DurationType, _ := each.Getstring("duration_type")
		Quantity, _ := each.Getint("quantity")
		FoodInstruction, _ := each.Getstring("food_instruction")
		morning, _ := each.Getfloat("morning")
		afternoon, _ := each.Getfloat("afternoon")
		night, _ := each.Getfloat("night")
		medicineArray = append(medicineArray, dto.MedicineArray{
			MedicineID:      MedicineID,
			DurationDay:     DurationDay,
			DurationType:    DurationType,
			Quantity:        Quantity,
			FoodInstruction: FoodInstruction,
			Morning:         morning,
			Afternoon:       afternoon,
			Night:           night,
		})
	}
	return medicineArray
}

func (Presc *PrescriptionController) FindMedicineDetInfo(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	prescriptionID := c.Params("prescription_id")
	if prescriptionID == "" {
		logger.Warn("prescription medicine info request invalid", zap.String("reason", "missing_prescription_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("prescription medicine info attempt", zap.String("prescription_id", prescriptionID))

	medicineInfoData, totalCount, err := Presc.PItemService.GetMedicineInfo(logger, prescriptionID)
	if err != nil {
		logger.Error("prescription medicine info response failed",
			zap.String("prescription_id", prescriptionID),
			zap.Error(err),
		)
		return wrapError.Wrap(err, c, fiber.StatusInternalServerError)
	}

	var response dto.Response
	response.Data = medicineInfoData
	response.Total = int(totalCount)
	response.Code = "200"
	response.Message = "fetched data successfully"
	return c.Status(fiber.StatusOK).JSON(response)
}
