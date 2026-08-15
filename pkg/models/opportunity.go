package models

// Opportunity is the immutable financial snapshot shown to the reseller.
// Amounts are stored in cents to avoid floating-point money calculations.
type Opportunity struct {
	Item                 Item    `json:"item"`
	ExpectedResaleCents  int64   `json:"expected_resale_cents"`
	ExpectedProfitCents  int64   `json:"expected_profit_cents"`
	MaximumPurchaseCents int64   `json:"maximum_purchase_cents"`
	ApplicableCostsCents int64   `json:"applicable_costs_cents"`
	ROIPercent           float64 `json:"roi_percent"`
	Condition            string  `json:"condition"`
	Confidence           int     `json:"confidence"`
	EstimateMethod       string  `json:"estimate_method"`
}
