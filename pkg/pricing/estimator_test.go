package pricing

import (
	"testing"

	"github.com/2spy/vinted-discord-bot/pkg/history"
)

func TestEstimatorUsesSegmentWhenEnoughData(t *testing.T) {
	estimator := Estimator{MinimumModelData: 2, MinimumSegmentData: 2, Sales: []history.Sale{
		{Model: "Nike P-6000", Size: "42", Condition: "good", SaleCents: 3000},
		{Model: "Nike P-6000", Size: "42", Condition: "good", SaleCents: 3500},
		{Model: "Nike P-6000", Size: "43", Condition: "good", SaleCents: 6000},
	}}
	got, ok := estimator.Estimate("Nike P-6000", "42", "good")
	if !ok || got.ExpectedCents != 3250 || got.Method != "historical_segment" {
		t.Fatalf("unexpected estimate: %+v, %v", got, ok)
	}
}

func TestEstimatorFallsBackWhenDataIsInsufficient(t *testing.T) {
	estimator := Estimator{MinimumModelData: 3, MinimumSegmentData: 2, Fallback: map[string]Estimate{"nike p-6000": {ExpectedCents: 3500, Method: "manual_fallback"}}}
	got, ok := estimator.Estimate("Nike P-6000", "42", "good")
	if !ok || got.ExpectedCents != 3500 || got.Method != "manual_fallback" {
		t.Fatalf("unexpected fallback: %+v, %v", got, ok)
	}
}
