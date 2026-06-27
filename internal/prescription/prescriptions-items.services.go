package prescription

import (
	"context"
	"hospital-backend/internal/prescription/dto"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PrescriptionItemServ struct {
	PrescRepo PrescItemsRepo
}

func NewPrescriptionItemService(PItems PrescItemsRepo) *PrescriptionItemServ {
	return &PrescriptionItemServ{PrescRepo: PItems}
}
func (s *PrescriptionItemServ) AddItems(db *gorm.DB, medicine []dto.MedicineArray, prescriptionID string, userID string) (err error) {
	prescriptionItems := s.toPrescItems(medicine, prescriptionID, userID)
	err = s.PrescRepo.AddItems(db, prescriptionItems)
	if err != nil {
		return
	}
	return nil
}
func (s *PrescriptionItemServ) toPrescItems(med []dto.MedicineArray, pID string, userID string) []PrescriptionItems {
	var prescItems []PrescriptionItems

	for _, each := range med {
		var pItem PrescriptionItems
		pItem.ID = uuid.New().String()
		pItem.MedicineID = each.MedicineID
		pItem.FoodInstruction = each.FoodInstruction
		pItem.Frequency.Night = each.Night
		pItem.Frequency.Morning = each.Morning
		pItem.Frequency.Afternoon = each.Afternoon
		pItem.DurationDay = each.DurationDay
		pItem.DurationType = s.parseDurationtype(each.DurationType)
		pItem.Quantity = s.calculateQuantity(pItem.Frequency, int(each.DurationDay), each.DurationType)
		pItem.BalanceAfterDispense = 0
		pItem.PrescriptionID = pID
		pItem.Status = StatusPending
		pItem.CreatedAt = time.Now()
		pItem.CreatedBy = userID
		prescItems = append(prescItems, pItem)
	}
	return prescItems
}
func (s *PrescriptionItemServ) parseDurationtype(durationtype string) string {
	switch durationtype {
	case Days:
		return Days
	case Weeks:
		return Weeks
	case Month:
		return Month
	default:
		return Days
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
	case Days:
		qty = durationDay * count
	case Weeks:
		qty = durationDay * count * 7
	case Month:
		qty = durationDay * count * 30
	}
	return qty
}
func (s *PrescriptionItemServ) GetPrescriptionsByPID(pID string, limit float64, pageno float64) ([]MixedPrescriptionItem, int64, error) {
	query := `select p.id as prescription_id, p.frequency,p.duration_day,p.duration_type,p.quantity,p.food_instruction,m.id as medicine_id, m.name as medicine_name,m.form as medicine_form,m.strength as medicine_strength 
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
                    minv.shelf_location
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
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	medicineDet, err := p.PrescRepo.FindMedicineInfoByPID(ctx, query, prescriptionID)
	if err != nil {
		return nil, err
	}
	//unit selling price calculation
	return medicineDet, nil

}
