package patient

import (
	"hospital-backend/internal/organisation"
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Rooms struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RoomType   string    `json:"room_type" gorm:"not null"`
	WardNumber string    `json:"ward_number" gorm:"not null"`
	Status     string    `json:"status" gorm:"default:'available'"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
type Bed struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RoomID      string    `json:"room_id" gorm:"type:uuid;not null"`
	Status      string    `json:"status" gorm:"default:'available'"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	PricePerDay float64   `json:"price_per_day"`
}

type Patient struct {
	ID              string                    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name            string                    `json:"name" gorm:"not null"`
	Age             int                       `json:"age" gorm:"not null"`
	Gender          string                    `json:"gender" gorm:"not null"`
	Weight          int                       `json:"weight" gorm:"not null"`
	AdmissionDate   time.Time                 `json:"admission_date" gorm:"not null"`
	DischargeDate   *time.Time                `json:"discharge_date"`
	EmailID         string                    `json:"email_id" gorm:"unique"`
	MobileNumber    string                    `json:"mobile_no" gorm:"unique"`
	Symptoms        pq.StringArray            `json:"symptoms" gorm:"type:text[]"`
	ActiveCondition string                    `json:"active_condition" `
	CreatedBy       string                    `json:"created_by"`
	OrganisationID  string                    `json:"organisation_id"`
	Organisation    organisation.Organisation `gorm:"foreignKey:OrganisationID"`
	DoctorID        string                    `json:"doctor_id"`
}

type BedAllotment struct {
	ID             string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PatientID      string         `json:"patient_id" gorm:"type:uuid;not null"`
	BedID          string         `json:"bed_id" gorm:"type:uuid;not null"`
	RoomID         string         `json:"room_id" gorm:"type:uuid;not null"`
	Prescriptions  datatypes.JSON `json:"prescriptions" gorm:"type:jsonb"`
	EmergencyEntry bool           `json:"emergency_entry" gorm:"default:false"`
	FoodIncluded   bool           `json:"food_included" gorm:"default:false"`
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	AdmittedAt     time.Time      `json:"admitted_at" gorm:"autoCreateTime"`
	DischargedAt   *time.Time     `json:"discharged_at"`
	CreatedBy      string         `json:"created_by"`
}

/*
Operations Flow: Bed Allotment & Patient Stay Management

1️⃣ Room Selection
   - User (receptionist or system) selects preferred room type (e.g., Deluxe, General, ICU, Ward).
   - System fetches only available beds of that type.
   - If multiple beds available → random or least-occupied bed is auto-assigned.
   - If no beds available → add patient to waiting list or show “No Availability”.

2️⃣ Bed Allotment
   - Create a new BedAllotment record:
        → PatientID
        → BedID
        → RoomType
        → AdmittedAt (auto)
        → FoodIncluded (true/false based on hospital or patient choice)
        → EmergencyEntry (true if admitted via emergency ward)
   - Update Bed.status = "occupied"

3️⃣ Prescriptions & Treatments
   - Nurse or doctor can add prescriptions during stay.
   - Prescription table logs:
        → BedAllotmentID (foreign key)
        → PrescribedBy (doctor/nurse)
        → Medicine name, dosage, frequency, notes
        → Event types (blood given, saline, routine BP check, etc.)
   - Supports multiple prescriptions per stay.
   - Helps maintain audit trail and patient care continuity.

4️⃣ Bed Transfer / ICU Shift
   - If patient condition changes:
        → Discharge current bed allotment (partial discharge with reason “ICU Shift”)
        → Create new BedAllotment record for ICU room.
        → Copy relevant prescriptions or continue treatment chain.

5️⃣ Discharge
   - On discharge:
        → Set DischargedAt = time.Now()
        → Update Bed.status = "available"
        → Mark all active prescriptions as completed/closed
        → Generate billing summary (room cost + food + extra meds)

6️⃣ Food & Accommodation
   - FoodIncluded flag default true for premium packages.
   - Daily food charges or meal logs can be added for billing integration.

7️⃣ Emergency & Alerts
   - EmergencyEntry = true triggers priority room assignment.
   - If patient’s condition deteriorates, trigger alerts for staff shift or ICU bed lookup.

8️⃣ Reporting / Audit
   - Track bed utilization by room type.
   - Track total prescriptions per patient stay.
   - Analyze nurse activity logs.
   - Generate discharge summary for doctors/patients.

*/
