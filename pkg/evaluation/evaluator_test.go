package evaluation

import (
	"testing"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

func TestMatchModel(t *testing.T) {
	model, err := MatchModel("Nike P 6000 Metallic Silver")
	if err != nil || model != "Nike P-6000" {
		t.Fatalf("unexpected match: %q, %v", model, err)
	}
}

func TestEvaluateIncludesExactMinimumProfit(t *testing.T) {
	result := Evaluate(models.Item{Title: "Nike P-6000", Brand: "Nike", Size: "42", Price: 15}, PricePolicy{
		ExpectedResaleCents: 3500,
		MinimumProfitCents:  1300,
	}, ConditionGood)
	if !result.Qualified || result.ExpectedProfitCents != 2000 || result.MaximumPurchaseCents != 2200 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestEvaluateRejectsUnknownCondition(t *testing.T) {
	result := Evaluate(models.Item{Title: "Nike P-6000", Brand: "Nike", Size: "42", Price: 10}, PricePolicy{
		ExpectedResaleCents: 3500,
		MinimumProfitCents:  1300,
	}, ConditionUnknown)
	if result.Qualified || result.Reason != "condition is not acceptable or not assessed" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
