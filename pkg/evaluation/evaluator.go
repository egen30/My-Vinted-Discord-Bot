// Package evaluation contains the deterministic, auditable opportunity rules.
package evaluation

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

type Condition string

const (
	ConditionVeryGood Condition = "very_good"
	ConditionGood     Condition = "good"
	ConditionOkay     Condition = "okay"
	ConditionPoor     Condition = "poor"
	ConditionUnknown  Condition = "unknown"
)

type PricePolicy struct {
	ExpectedResaleCents int64
	ApplicableCostsCents int64
	MinimumProfitCents  int64
}

type Result struct {
	Qualified             bool
	Reason                string
	CanonicalModel        string
	PurchasePriceCents    int64
	ExpectedResaleCents   int64
	ApplicableCostsCents  int64
	MinimumProfitCents    int64
	ExpectedProfitCents   int64
	MaximumPurchaseCents  int64
	ROIPercent            float64
	Condition             Condition
}

var modelAliases = map[string][]string{
	"Nike P-6000":    {"p-6000", "p 6000", "p6000"},
	"Nike Air Max 95": {"air max 95", "airmax 95", "am95"},
	"Nike Air Max 90": {"air max 90", "airmax 90", "am90"},
	"Nike Air Max 97": {"air max 97", "airmax 97", "am97"},
	"Nike Shox":       {"shox"},
}

var whitespace = regexp.MustCompile(`\s+`)

// MatchModel returns a canonical target model, or an error for unmatched or ambiguous text.
func MatchModel(text string) (string, error) {
	normalized := strings.ToLower(whitespace.ReplaceAllString(strings.TrimSpace(text), " "))
	var matches []string
	for canonical, aliases := range modelAliases {
		for _, alias := range aliases {
			if strings.Contains(normalized, alias) {
				matches = append(matches, canonical)
				break
			}
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("target model not found")
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous target model: %s", strings.Join(matches, ", "))
	}
	return matches[0], nil
}

// Evaluate applies the €13-style profit gate to one listing and records its reason.
func Evaluate(item models.Item, policy PricePolicy, condition Condition) Result {
	purchase := int64(math.Round(item.Price * 100))
	result := Result{
		PurchasePriceCents:   purchase,
		ExpectedResaleCents:  policy.ExpectedResaleCents,
		ApplicableCostsCents: policy.ApplicableCostsCents,
		MinimumProfitCents:   policy.MinimumProfitCents,
		Condition:            condition,
	}
	model, err := MatchModel(item.Brand + " " + item.Title)
	if err != nil {
		result.Reason = err.Error()
		return result
	}
	result.CanonicalModel = model
	if strings.TrimSpace(item.Size) == "" {
		result.Reason = "EU size is missing"
		return result
	}
	if condition == ConditionPoor || condition == ConditionUnknown || condition == "" {
		result.Reason = "condition is not acceptable or not assessed"
		return result
	}
	result.ExpectedProfitCents = policy.ExpectedResaleCents - purchase - policy.ApplicableCostsCents
	result.MaximumPurchaseCents = policy.ExpectedResaleCents - policy.ApplicableCostsCents - policy.MinimumProfitCents
	if purchase+policy.ApplicableCostsCents > 0 {
		result.ROIPercent = float64(result.ExpectedProfitCents) / float64(purchase+policy.ApplicableCostsCents) * 100
	}
	if result.ExpectedProfitCents < policy.MinimumProfitCents {
		result.Reason = "expected profit is below minimum"
		return result
	}
	result.Qualified = true
	result.Reason = "meets target model, size, condition, and profit policy"
	return result
}
