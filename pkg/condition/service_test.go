package condition

import (
	"context"
	"testing"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

type fakeProvider struct {
	assessment Assessment
	err        error
}

func (f fakeProvider) Analyze(context.Context, Input) (Assessment, error) {
	return f.assessment, f.err
}

func TestServiceRejectsLowConfidence(t *testing.T) {
	service := Service{Provider: fakeProvider{assessment: Assessment{Rating: Good, Confidence: 0.4}}, MinConfidence: 0.75}
	assessment := service.Analyze(context.Background(), models.Item{ImageURL: "https://img.test/shoe"})
	if assessment.Rating != Unknown || assessment.Confidence != 0.4 {
		t.Fatalf("expected unknown low-confidence assessment, got %+v", assessment)
	}
}

func TestServiceUsesSafeUnknownWithoutProvider(t *testing.T) {
	assessment := (Service{}).Analyze(context.Background(), models.Item{ImageURL: "https://img.test/shoe"})
	if assessment.Rating != Unknown || len(assessment.Signals) == 0 {
		t.Fatalf("expected explicit unknown assessment, got %+v", assessment)
	}
}
