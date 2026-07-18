package prescription

import (
	"context"
	"errors"
	"fmt"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/pkg/constants"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrMedicineAlreadyPresent = errors.New("medicine already present in prescription")

type PrescriptionItemServ struct {
	PrescRepo PrescItemsRepo
}

func NewPrescriptionItemService(PItems PrescItemsRepo) *PrescriptionItemServ {
	return &PrescriptionItemServ{PrescRepo: PItems}
}
func (s *PrescriptionItemServ) AddItems(db *gorm.DB, medicine []dto.MedicineArray, prescriptionID string, userID string) (err error) {
	prescriptionItems, err := s.toPrescItems(db, medicine, prescriptionID, userID)
	if err != nil {
		return err
	}
	err = s.PrescRepo.AddItems(db, prescriptionItems)
	if err != nil {
		return err
	}
	return nil
}
func (s *PrescriptionItemServ) toPrescItems(db *gorm.DB, med []dto.MedicineArray, pID string, userID string) ([]PrescriptionItems, error) {
	var prescItems []PrescriptionItems
	medicineIDs, err := s.PrescRepo.GetMedicineIDsByPrescriptionID(db, pID)
	if err != nil {
		return nil, err
	}
	existingMedicines := make(map[string]struct{}, len(medicineIDs)+len(med))
	for _, medicineID := range medicineIDs {
		existingMedicines[medicineID] = struct{}{}
	}

	for _, each := range med {
		if _, exists := existingMedicines[each.MedicineID]; exists {
			return nil, ErrMedicineAlreadyPresent
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
		pItem.BalanceAfterDispense = 0
		pItem.PrescriptionID = pID
		pItem.Status = constants.StatusPending
		pItem.CreatedAt = time.Now()
		pItem.CreatedBy = userID
		prescItems = append(prescItems, pItem)
	}
	return prescItems, nil
}
func (s *PrescriptionItemServ) UpdatePrescriptionItemByID(req dto.UpdatePrescriptionItemRequest) error {
	existing, err := s.PrescRepo.GetPrescriptionItemByID(req.PrescriptionItemID)
	if err != nil {
		return err
	}
	// don't allow editing once dispensing has started, otherwise it desyncs with invoices
	if existing.BalanceAfterDispense > 0 ||
		existing.Status == constants.StatusFullyDispensed ||
		existing.Status == constants.StatusPartiallyDispensed {
		return fmt.Errorf("cannot edit prescription item %s: it has already been dispensed", req.PrescriptionItemID)
	}

	var item PrescriptionItems
	item.ID = req.PrescriptionItemID
	item.MedicineID = req.MedicineID
	item.Frequency = Freq{Morning: req.Morning, Afternoon: req.Afternoon, Night: req.Night}
	item.DurationDay = req.DurationDay
	item.DurationType = s.parseDurationtype(req.DurationType)
	item.FoodInstruction = req.FoodInstruction
	item.Quantity = int64(s.calculateQuantity(item.Frequency, int(req.DurationDay), item.DurationType))
	item.UpdatedAt = time.Now()

	return s.PrescRepo.UpdatePrescriptionItem(item)
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
func (s *PrescriptionItemServ) GetPrescriptionsByPIDWithLimit(pID string, limit float64, pageno float64) ([]MixedPrescriptionItem, int64, error) {
	query := `select p.id as prescription_item_id, p.frequency,p.duration_day,p.duration_type,p.quantity,p.food_instruction,m.id as medicine_id, 
	m.name as medicine_name,m.form as medicine_form, m.strength as medicine_strength 
	from prescription_items p
	join medicines m on p.medicine_id = m.id
	where p.prescription_id = $1
	limit $2
	offset $3`
	dblimit, dbpageno := s.parsePagination(limit, pageno)
	MixedResponse, err := s.PrescRepo.GetItemsByPrescriptionID(query, pID, dblimit, dbpageno)
	if err != nil {
		return nil, 0, err
	}

	totalCount, err := s.PrescRepo.GetTotalCountByPrescID(pID)
	if err != nil {
		return nil, 0, err
	}
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
func (p *PrescriptionItemServ) getMedicineInfo(prescriptionID string) ([]MedicineDetInfo, error) {
	query := `SELECT 
    p.code AS prescription_code,
    p.status AS prescription_status,
    p.created_at AS prescription_created_at,
    pI.prescription_id,
    pI.id AS prescription_item_id,
	pI.quantity AS prescribed_quantity,
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
	medicineDet, err := p.PrescRepo.FindMedicineInfoByPID(context.TODO(), query, prescriptionID)
	if err != nil {
		return nil, err
	}
	//unit selling price calculation
	return medicineDet, nil

}
func (p *PrescriptionItemServ) GetqtyByMedicine(prescriptionID string) (map[string]dto.PrescriptionQtyInfo, error) {
	prescriptionItems, err := p.PrescRepo.GetQtyInfoByMed(prescriptionID)
	if err != nil {
		return nil, err
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
	return prescriptionMap, nil
}
func (p *PrescriptionItemServ) UpdateDispenseItemQty(tx *gorm.DB, prescriptionItemID string, dispensedQty int64) error {
	query := "UPDATE prescription_items SET balance_after_dispense = balance_after_dispense + ? WHERE id = ?"
	return p.PrescRepo.UpdateDispenseItemQty(tx, query, prescriptionItemID, dispensedQty)
}
func (p *PrescriptionItemServ) UpdateIPrescriptionStatus(tx *gorm.DB, prescriptionItemID string, status string) error {
	var item PrescriptionItems
	item.ID = prescriptionItemID
	item.Status = status
	item.UpdatedAt = time.Now()
	err := p.PrescRepo.UpdatePrescriptionItemStatus(tx, item)
	if err != nil {
		return err
	}
	return nil
}
func (p *PrescriptionItemServ) GetPrescriptionItemsByPID(pID string) ([]MixedPrescriptionItem, error) {
	query := `select p.id as prescription_id, p.frequency,p.duration_day,p.duration_type, p.quantity,p.food_instruction,m.id as medicine_id, m.name as medicine_name,m.form as medicine_form,
	m.strength as medicine_strength 
	from prescription_items p
	join medicines m on p.medicine_id = m.id
	where p.prescription_id = $1`
	prescriptionItems, err := p.PrescRepo.GetItemsByPrescriptionID(query, pID)
	if err != nil {
		return nil, err
	}
	return prescriptionItems, nil
}
