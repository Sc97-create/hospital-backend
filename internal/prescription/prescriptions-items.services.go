package prescription

import (
	"context"
	"errors"
	"fmt"
	patientdto "hospital-backend/internal/patient/dto"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PrescriptionItemServ struct {
	PrescRepo PrescItemsRepo
}

func NewPrescriptionItemService(PItems PrescItemsRepo) *PrescriptionItemServ {
	return &PrescriptionItemServ{PrescRepo: PItems}
}

func (s *PrescriptionItemServ) AddItems(log *zap.Logger, db *gorm.DB, medicine []dto.MedicineArray, prescriptionID string, userID string) (err error) {
	log = ensureLog(log)
	prescriptionItems, err := s.toPrescItems(log, db, medicine, prescriptionID, userID)
	if err != nil {
		return err
	}
	err = s.PrescRepo.AddItems(log, db, prescriptionItems)
	if err != nil {
		return err
	}
	return nil
}

func (s *PrescriptionItemServ) toPrescItems(log *zap.Logger, db *gorm.DB, med []dto.MedicineArray, pID string, userID string) ([]PrescriptionItems, error) {
	log = ensureLog(log)
	var prescItems []PrescriptionItems
	medicineIDs, err := s.PrescRepo.GetMedicineIDsByPrescriptionID(log, db, pID)
	if err != nil {
		return nil, err
	}
	existingMedicines := make(map[string]struct{}, len(medicineIDs)+len(med))
	for _, medicineID := range medicineIDs {
		existingMedicines[medicineID] = struct{}{}
	}

	for _, each := range med {
		if each.MedicineID == "" {
			return nil, &validationError{Field: "medicine_id", Msg: "medicine_id is required"}
		}
		if _, exists := existingMedicines[each.MedicineID]; exists {
			return nil, wrapError.ErrMedicineAlreadyPresent
		}
		existingMedicines[each.MedicineID] = struct{}{}

		var pItem PrescriptionItems
		pItem.ID = uuid.New().String()
		pItem.MedicineID = each.MedicineID
		pItem.FoodInstruction = each.FoodInstruction
		pItem.Frequency.Night = each.Night
		pItem.Frequency.Morning = each.Morning
		pItem.Frequency.Afternoon = each.Afternoon
		pItem.DurationDay = each.DurationDay
		pItem.DurationType = s.parseDurationtype(each.DurationType)
		pItem.Quantity = int64(s.calculateQuantity(pItem.Frequency, int(each.DurationDay), each.DurationType))
		pItem.BalanceAfterDispense = int(pItem.Quantity)
		pItem.PrescriptionID = pID
		pItem.Status = constants.StatusPending
		pItem.OutOfStock = false
		pItem.CreatedAt = time.Now()
		pItem.CreatedBy = userID
		prescItems = append(prescItems, pItem)
	}
	return prescItems, nil
}

type validationError struct {
	Field string
	Msg   string
}

func (e *validationError) Error() string {
	return e.Msg
}

func (s *PrescriptionItemServ) UpdatePrescriptionItemByID(log *zap.Logger, req dto.UpdatePrescriptionItemRequest) error {
	log = ensureLog(log)
	existing, err := s.PrescRepo.GetPrescriptionItemByID(log, req.PrescriptionItemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("prescription item update failed",
				zap.String("prescription_item_id", req.PrescriptionItemID),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrPrescriptionItemNotFound
		}
		log.Error("prescription item update failed",
			zap.String("prescription_item_id", req.PrescriptionItemID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return wrapError.ErrPrescriptionItemUpdateFailed
	}

	if existing.BalanceAfterDispense < int(existing.Quantity) ||
		existing.Status == constants.StatusFullyDispensed ||
		existing.Status == constants.StatusPartiallyDispensed {
		log.Warn("prescription item update failed",
			zap.String("prescription_item_id", req.PrescriptionItemID),
			zap.String("reason", "already_dispensed"),
			zap.String("status", existing.Status),
		)
		return wrapError.ErrCannotEditDispensedItem
	}

	var item PrescriptionItems
	item.ID = req.PrescriptionItemID
	item.MedicineID = req.MedicineID
	item.Frequency = Freq{Morning: req.Morning, Afternoon: req.Afternoon, Night: req.Night}
	item.DurationDay = req.DurationDay
	item.DurationType = s.parseDurationtype(req.DurationType)
	item.FoodInstruction = req.FoodInstruction
	item.Quantity = int64(s.calculateQuantity(item.Frequency, int(req.DurationDay), item.DurationType))
	item.BalanceAfterDispense = int(item.Quantity)
	item.UpdatedAt = time.Now()

	if err = s.PrescRepo.UpdatePrescriptionItem(log, item); err != nil {
		log.Error("prescription item update failed",
			zap.String("prescription_item_id", req.PrescriptionItemID),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrPrescriptionItemUpdateFailed
	}

	log.Info("prescription item update success",
		zap.String("prescription_item_id", req.PrescriptionItemID),
		zap.String("medicine_id", req.MedicineID),
	)
	return nil
}

func (s *PrescriptionItemServ) parseDurationtype(durationtype string) string {
	switch durationtype {
	case constants.Days:
		return constants.Days
	case constants.Weeks:
		return constants.Weeks
	case constants.Month:
		return constants.Month
	default:
		return constants.Days
	}
}

func (s *PrescriptionItemServ) calculateQuantity(freq Freq, durationDay int, durationtype string) int {
	count := 0
	if freq.Morning != 0 {
		count++
	}
	if freq.Afternoon != 0 {
		count++
	}
	if freq.Night != 0 {
		count++
	}
	var qty int
	switch durationtype {
	case constants.Days:
		qty = durationDay * count
	case constants.Weeks:
		qty = durationDay * count * 7
	case constants.Month:
		qty = durationDay * count * 30
	}
	return qty
}

func (s *PrescriptionItemServ) GetPrescriptionsByPIDWithLimit(log *zap.Logger, pID string, limit float64, pageno float64) ([]MixedPrescriptionItem, int64, error) {
	log = ensureLog(log)
	query := `select p.id as prescription_item_id, p.frequency,p.duration_day,p.duration_type,p.quantity,p.food_instruction,m.id as medicine_id, 
	m.name as medicine_name,m.form as medicine_form, m.strength as medicine_strength 
	from prescription_items p
	join medicines m on p.medicine_id = m.id
	where p.prescription_id = $1
	limit $2
	offset $3`
	dblimit, dbpageno := s.parsePagination(limit, pageno)
	MixedResponse, err := s.PrescRepo.GetItemsByPrescriptionID(log, query, pID, dblimit, dbpageno)
	if err != nil {
		log.Error("prescription items get failed",
			zap.String("prescription_id", pID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrPrescriptionFetchFailed
	}

	totalCount, err := s.PrescRepo.GetTotalCountByPrescID(log, pID)
	if err != nil {
		log.Error("prescription items get failed",
			zap.String("prescription_id", pID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrPrescriptionFetchFailed
	}

	log.Info("prescription items get success",
		zap.String("prescription_id", pID),
		zap.Int("count", len(MixedResponse)),
		zap.Int64("total", totalCount),
	)
	return MixedResponse, totalCount, nil
}

func (s *PrescriptionItemServ) parsePagination(limit float64, pageno float64) (int, int) {
	numLimit := int(limit)
	numpageno := int(pageno)
	skip := 0
	if numpageno != 0 {
		skip = (numpageno - 1) * numLimit
	}
	return numLimit, skip
}

func (p *PrescriptionItemServ) GetMedicineInfo(log *zap.Logger, prescriptionID string) (MedicineInfoResult, error) {
	log = ensureLog(log)
	medicineDet, err := p.fetchMedicineDetails(log, prescriptionID)
	if err != nil {
		return MedicineInfoResult{}, err
	}

	totalCount, err := p.PrescRepo.GetTotalCountByPrescID(log, prescriptionID)
	if err != nil {
		log.Error("prescription medicine info failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return MedicineInfoResult{}, wrapError.ErrMedicineInfoFetchFailed
	}

	patientData, err := p.fetchPatientByPrescriptionID(log, prescriptionID)
	if err != nil {
		return MedicineInfoResult{}, err
	}

	log.Info("prescription medicine info success",
		zap.String("prescription_id", prescriptionID),
		zap.Int("count", len(medicineDet)),
		zap.Int64("total", totalCount),
	)
	return MedicineInfoResult{
		Items:       medicineDet,
		PatientData: patientData,
		Total:       totalCount,
	}, nil
}

func (p *PrescriptionItemServ) fetchMedicineDetails(log *zap.Logger, prescriptionID string) ([]MedicineDetInfo, error) {
	query := `SELECT 
    p.code AS prescription_code,
    p.status AS prescription_status,
    p.created_at AS prescription_created_at,
    pI.prescription_id,
    pI.id AS prescription_item_id,
	pI.quantity AS prescribed_quantity,
	pI.balance_after_dispense AS remaining_quantity,
	pI.status AS prescription_item_status,
	pI.out_of_stock AS out_of_stock,
	pI.food_instruction AS food_instruction,
    m.id AS medicine_id,
    m.name AS medicine_name,
    m.form AS medicine_form,
    m.strength AS medicine_strength,
    pI.frequency,
    m.reorder_level,
    m.max_stock_target,
    COALESCE(
        (
            SELECT json_agg(batch_data ORDER BY batch_data.expires_at ASC)
            FROM (
                SELECT 
                    minv.id AS batch_id,
                    minv.batch_no,
                    minv.expires_at,
                    minv.current_stock_units,
                    minv.units_per_box,
                    minv.pricing,
                    minv.shelf_location,
					minv.supplier_id
                FROM medicine_inventories minv
                WHERE minv.medicine_id = m.id
                  AND minv.current_stock_units > 0     
                  AND minv.expires_at > CURRENT_DATE   
            ) batch_data
        ), 
        '[]'::json
    ) AS medicine_batches
FROM prescription_items pI
JOIN prescriptions p ON pI.prescription_id = p.id
JOIN medicines m ON pI.medicine_id = m.id
WHERE pI.prescription_id = $1;
	`
	medicineDet, err := p.PrescRepo.FindMedicineInfoByPID(log, context.TODO(), query, prescriptionID)
	if err != nil {
		log.Error("prescription medicine info failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, wrapError.ErrMedicineInfoFetchFailed
	}
	return medicineDet, nil
}

func (p *PrescriptionItemServ) fetchPatientByPrescriptionID(log *zap.Logger, prescriptionID string) (patientdto.PatientResponse, error) {
	query := `SELECT
    pa.id AS patient_id,
    pa.uh_id AS patient_code,
    pa.name AS patient_name,
    pa.weight AS patient_weight,
    pa.gender AS patient_gender,
    pa.mobile_number AS patient_phone,
    pa.address AS patient_address,
    pa.email_id AS patient_email,
    pa.status AS patient_status,
    pa.age AS patient_age,
    pa.blood_group AS patient_bg,
    pa.last_visit_date AS patient_lvd,
    pa.created_at AS patient_created_at
FROM prescriptions p
JOIN patients pa ON pa.id = p.patient_id
WHERE p.id = $1`
	patientRow, err := p.PrescRepo.GetPatientByPrescriptionID(log, query, prescriptionID)
	if err != nil {
		log.Error("prescription medicine info failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("reason", "patient_db_read"),
			zap.Error(err),
		)
		return patientdto.PatientResponse{}, wrapError.ErrMedicineInfoFetchFailed
	}
	return mapMedicineInfoPatient(patientRow), nil
}

func mapMedicineInfoPatient(row MedicineInfoPatientRow) patientdto.PatientResponse {
	return patientdto.PatientResponse{
		PatientID:        row.PatientID,
		PatientCode:      row.PatientCode,
		PatientName:      row.PatientName,
		PatientWeight:    row.PatientWeight,
		PatientGender:    row.PatientGender,
		PatientPhone:     row.PatientPhone,
		PatientAddress:   row.PatientAddress,
		PatientEmail:     row.PatientEmail,
		PatientImage:     "",
		PatientStatus:    row.PatientStatus,
		PatientAge:       row.PatientAge,
		PatientBG:        row.PatientBG,
		PatientLVD:       row.PatientLVD,
		WaitingTime:      formatWaitingTime(row.PatientLVD),
		PatientCreatedAt: row.PatientCreatedAt,
	}
}

func formatWaitingTime(lastVisit time.Time) string {
	if lastVisit.IsZero() {
		return "0"
	}
	duration := time.Since(lastVisit)
	minutes := duration.Minutes()
	hours := duration.Hours()
	days := hours / 24
	if days >= 30 {
		return "0"
	}
	if hours >= 24 {
		return fmt.Sprintf("%.0f days", days)
	}
	if minutes >= 60 {
		return fmt.Sprintf("%.0f hrs", hours)
	}
	return fmt.Sprintf("%.0f mins", minutes)
}

func (p *PrescriptionItemServ) GetqtyByMedicine(log *zap.Logger, prescriptionID string) (map[string]dto.PrescriptionQtyInfo, error) {
	log = ensureLog(log)
	prescriptionItems, err := p.PrescRepo.GetQtyInfoByMed(log, prescriptionID)
	if err != nil {
		log.Error("prescription qty map failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, wrapError.ErrPrescriptionFetchFailed
	}
	prescriptionMap := make(map[string]dto.PrescriptionQtyInfo)
	for _, each := range prescriptionItems {
		var eachPrescription dto.PrescriptionQtyInfo
		eachPrescription.MedicineID = each.MedicineID
		eachPrescription.Quantity = each.Quantity
		eachPrescription.PrescriptionID = each.ID
		eachPrescription.BalanceAfterDispense = each.BalanceAfterDispense
		prescriptionMap[each.MedicineID] = eachPrescription
	}
	log.Debug("prescription qty map success",
		zap.String("prescription_id", prescriptionID),
		zap.Int("medicine_count", len(prescriptionMap)),
	)
	return prescriptionMap, nil
}

func (p *PrescriptionItemServ) UpdateDispenseItemQty(log *zap.Logger, tx *gorm.DB, prescriptionItemID string, dispensedQty int64) error {
	log = ensureLog(log)
	query := "UPDATE prescription_items SET balance_after_dispense = balance_after_dispense - ? WHERE id = ?"
	err := p.PrescRepo.UpdateDispenseItemQty(log, tx, query, prescriptionItemID, dispensedQty)
	if err != nil {
		log.Error("prescription item dispense qty update failed",
			zap.String("prescription_item_id", prescriptionItemID),
			zap.Int64("dispensed_qty", dispensedQty),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrPrescriptionItemUpdateFailed
	}
	log.Debug("prescription item dispense qty updated",
		zap.String("prescription_item_id", prescriptionItemID),
		zap.Int64("dispensed_qty", dispensedQty),
	)
	return nil
}

func (p *PrescriptionItemServ) UpdateIPrescriptionStatus(log *zap.Logger, tx *gorm.DB, prescriptionItemID string, status string, outOfStock bool) error {
	log = ensureLog(log)
	err := p.PrescRepo.UpdatePrescriptionItemStatus(log, tx, prescriptionItemID, status, outOfStock)
	if err != nil {
		log.Error("prescription item status update failed",
			zap.String("prescription_item_id", prescriptionItemID),
			zap.String("status", status),
			zap.Bool("out_of_stock", outOfStock),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrPrescriptionItemUpdateFailed
	}
	log.Info("prescription item status update success",
		zap.String("prescription_item_id", prescriptionItemID),
		zap.String("status", status),
		zap.Bool("out_of_stock", outOfStock),
	)
	return nil
}

func (p *PrescriptionItemServ) GetPrescriptionItemsByPID(log *zap.Logger, pID string) ([]MixedPrescriptionItem, error) {
	log = ensureLog(log)
	query := `select p.id as prescription_id, p.frequency,p.duration_day,p.duration_type, p.quantity,p.food_instruction,m.id as medicine_id, m.name as medicine_name,m.form as medicine_form,
	m.strength as medicine_strength 
	from prescription_items p
	join medicines m on p.medicine_id = m.id
	where p.prescription_id = $1`
	prescriptionItems, err := p.PrescRepo.GetItemsByPrescriptionID(log, query, pID)
	if err != nil {
		log.Error("prescription items get failed",
			zap.String("prescription_id", pID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, wrapError.ErrPrescriptionFetchFailed
	}
	return prescriptionItems, nil
}
