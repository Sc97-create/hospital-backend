package dto

// PlanDetail is the catalog row returned to the frontend for plan selection.
type PlanDetail struct {
	ID                    string         `json:"id"`
	Name                  string         `json:"name"`
	Status                string         `json:"status"`
	MonthlyPrice          float64        `json:"monthly_price"`
	Discount              float64        `json:"discount"`
	EffectiveMonthlyPrice float64        `json:"effective_monthly_price"`
	PlanDetails           map[string]any `json:"plan_details"`
}
