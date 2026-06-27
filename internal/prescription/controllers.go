package prescription

import (
	"errors"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/shared/params"

	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
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
	GetPrescriptionByID(c *fiber.Ctx) error
	GetPrescriptionsByPatientID(c *fiber.Ctx) error
	// GetPrescriptionsByDoctorID(c *fiber.Ctx) error
	AddPrescriptionItems(c *fiber.Ctx) error
	// DeletePrescription(c *fiber.Ctx) error
	FindMany(c *fiber.Ctx) error
	FindPrescriptionByID(c *fiber.Ctx) error
	UpdateStatus(c *fiber.Ctx) error
	FindMedicineDetInfo(c *fiber.Ctx) error
	DispenseMedicine(c *fiber.Ctx) error
	//FindPrescriptionByPatientID(c *fiber.Ctx) error
}

func (PresC *PrescriptionController) CreatePrescription(c *fiber.Ctx) error {
	payload, err := params.New(c)
	if err != nil {
		return err
	}
	requestmap := dto.CreatePrescriptionRequest{}
	requestmap.AppointmentID, err = payload.Getstring("appointment_id")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	requestmap.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	requestmap.PrescribedBy, err = payload.Getstring("prescribed_by")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}

	medicines, err := payload.GetChildren("medicine_array")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	requestmap.MedicineArray = PresC.toMedicineArray(medicines)
	id, err := PresC.PService.CreatePrescription(requestmap)
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	var response dto.CreatePrescriptionResponse
	response.Code = "200"
	response.Data = dto.Data{ID: id}
	response.Message = "prescription saved successfully"
	return c.Status(200).JSON(response)
}

func (PresC *PrescriptionController) FindMany(c *fiber.Ctx) error {
	var requestmap dto.FindManyRequest
	err := c.QueryParser(&requestmap)
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	prescriptions, totalcount, err := PresC.PService.FindMany(requestmap.Limit, requestmap.Offset, requestmap.OrganisationID)
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	var response dto.FindManyResponse
	response.Code = "200"
	response.Message = "prescriptions fetched successfully"
	response.Data = prescriptions
	response.TotalCount = totalcount
	return c.Status(200).JSON(response)
}

func (Presc *PrescriptionController) AddPrescriptionItems(c *fiber.Ctx) error {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	var requestMap dto.UpdateRequest
	requestMap.PrescriptionID, err = payload.Getstring("prescription_id")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	requestMap.UserID, err = payload.Getstring("prescribed_by")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	err = Presc.validateIDs(requestMap.PrescriptionID)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}

	medicineArr, err := payload.GetChildren("medicine_array")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	requestMap.MedicineArr = Presc.toMedicineArray(medicineArr)
	err = Presc.PService.AddPrescriptionItems(requestMap)
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	var response dto.CreatePrescriptionResponse
	response.Code = "200"
	response.Message = "prescription updated successfully"
	response.Data = dto.Data{ID: requestMap.PrescriptionID}
	return c.Status(200).JSON(response)
}
func (PresC *PrescriptionController) FindPrescriptionByID(c *fiber.Ctx) error {
	prescriptionID := c.Query("prescription_id")
	limit := c.QueryFloat("limit", 10)
	offset := c.QueryFloat("offset", 0)
	err := PresC.validateIDs(prescriptionID)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	medicines, totalCount, err := PresC.PItemService.GetPrescriptionsByPID(prescriptionID, limit, offset)
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	var response dto.FindPrescriptionByIDResponse
	response.Code = "200"
	response.Message = "prescription fetched successfully"
	response.Data.MedicineResponse = medicines
	response.Data.TotalCount = int(totalCount)
	//response.Data.CreatedAt = createdAt
	return c.Status(200).JSON(response)
}
func (PresC *PrescriptionController) UpdateStatus(c *fiber.Ctx) error {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	prescriptionID, err := payload.Getstring("prescription_id")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	appointmentID, err := payload.Getstring("appointment_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	err = PresC.PService.UpdateStatus(prescriptionID, appointmentID)
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	var response dto.CreatePrescriptionResponse
	response.Code = "200"
	response.Message = "prescription status updated successfully"
	response.Data = dto.Data{ID: prescriptionID}
	return c.Status(200).JSON(response)
}
func (PresC *PrescriptionController) GetPrescriptionByPatientID(c *fiber.Ctx) error {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	var reqmodel dto.PresPatients
	reqmodel.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqmodel.Pageno, err = payload.Getfloat("page_no")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqmodel.Limit, err = payload.Getfloat("limit")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqmodel.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	response, err := PresC.PService.GetPrescriptionByPatientID(reqmodel)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(response)
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
		//MedicineName, _ := each.Getstring("medicine_name")
		DurationDay, _ := each.Getfloat("duration")
		DurationType, _ := each.Getstring("duration_type")
		Quantity, _ := each.Getint("quantity")
		//MedicineType, _ := each.Getstring("medicine_type")
		FoodInstruction, _ := each.Getstring("food_instruction")
		morning, _ := each.Getfloat("morning")
		afternoon, _ := each.Getfloat("afternoon")
		night, _ := each.Getfloat("night")
		//dosage, _ := each.Getstring("dosage")
		medicineArray = append(medicineArray, dto.MedicineArray{
			MedicineID: MedicineID,
			//MedicineName:    MedicineName,
			DurationDay:  DurationDay,
			DurationType: DurationType,
			Quantity:     Quantity,
			//MedicineType:    MedicineType,
			FoodInstruction: FoodInstruction,
			Morning:         morning,
			Afternoon:       afternoon,
			Night:           night,
			//Dosage:          dosage,
		})

	}
	return medicineArray
}
func (Presc *PrescriptionController) FindMedicineDetInfo(c *fiber.Ctx) (err error) {
	prescriptionID := c.Params("prescription_id")
	medicineInfoData, err := Presc.PItemService.getMedicineInfo(prescriptionID)
	if err != nil {
		return
	}
	var response dto.Response
	response.Data = medicineInfoData
	response.Code = "200"

	return c.Status(200).JSON(response)
}
func (Presc *PrescriptionController) DispenseMedicine(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	var requestPayload dto.DispensePayload
	requestPayload.PrescriptionID, err = payload.Getstring("prescription_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	requestPayload.CashierID, err = payload.Getstring("cashier_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	requestPayload.SupplierID, err = payload.Getstring("supplier_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	requestPayload.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	requestPayload.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	requestPayload.PaymentMode, err = payload.Getstring("payment_mode")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	financials, err := payload.GetObject("financials")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	requestPayload.Financials, err = Presc.tomapfinancedto(financials)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	dispenseItems, err := payload.GetChildren("dispensed_items")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	requestPayload.DispensedItems, err = Presc.tomapDispense(dispenseItems)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}

	return
}
func (Presc *PrescriptionController) tomapfinancedto(finance *params.Payload) (dto.FinancialsDTO, error) {
	var financePayload dto.FinancialsDTO
	var err error
	financePayload.DiscountAmount, err = finance.Getfloat("discount_amount")
	if err != nil {
		return dto.FinancialsDTO{}, err
	}
	financePayload.SubtotalAmount, err = finance.Getfloat("subtotal_amount")
	if err != nil {
		return dto.FinancialsDTO{}, err
	}
	financePayload.TaxAmount, err = finance.Getfloat("tax_amount")
	if err != nil {
		return dto.FinancialsDTO{}, err
	}
	financePayload.TotalAmountPaid, err = finance.Getfloat("total_amount_paid")
	if err != nil {
		return dto.FinancialsDTO{}, err
	}
	return financePayload, nil
}
func (Presc *PrescriptionController) tomapDispense(dispensedItems []*params.Payload) ([]dto.DispensedItemDTO, error) {
	var dispensePayload []dto.DispensedItemDTO
	var err error
	for _, each := range dispensedItems {
		var eachMedicinedata dto.DispensedItemDTO
		eachMedicinedata.BatchID, err = each.Getstring("batch_id")
		if err != nil {
			continue
		}
		eachMedicinedata.MedicineID, err = each.Getstring("medicine_id")
		if err != nil {
			return nil, err
		}
		eachMedicinedata.ComputedItemTotal, err = each.Getfloat("computed_item_total") // need validation from baackend
		if err != nil {
			return nil, err
		}
		eachMedicinedata.QuantitySoldUnits, err = each.Getint("quantity_sold_units") // required to send notification
		if err != nil {
			return nil, err
		}
		eachMedicinedata.UnitPriceCharged, err = each.Getfloat("unit_price_chared")
		if err != nil {
			return nil, err
		}
		dispensePayload = append(dispensePayload, eachMedicinedata)
	}
	return dispensePayload, nil
}
