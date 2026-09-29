package plans

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Plan is a commercial product plan that subscriptions can reference.
type Plan struct {
	ID           string      `json:"id" gorm:"type:uuid;primaryKey"`
	Name         string      `json:"name" gorm:"type:varchar(255);not null;uniqueIndex"`
	Status       string      `json:"status" gorm:"type:text;not null"`
	MonthlyPrice float64     `json:"monthly_price" gorm:"type:numeric(12,2);not null;default:0"`
	Discount     float64     `json:"discount" gorm:"type:numeric(12,2);not null;default:0"`
	PlanDetails  PlanDetails `json:"plan_details" gorm:"type:jsonb"`
	CreatedAt    time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Plan) TableName() string {
	return "plans"
}

// PlanDetails is a free-form JSON object stored on the plan row.
type PlanDetails map[string]any

func (p *PlanDetails) Scan(value interface{}) error {
	if value == nil {
		*p = PlanDetails{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return errors.New("unsupported type for PlanDetails")
	}
}

func (p PlanDetails) Value() (driver.Value, error) {
	if p == nil {
		return json.Marshal(PlanDetails{})
	}
	return json.Marshal(p)
}
