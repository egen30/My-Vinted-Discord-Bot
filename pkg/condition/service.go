// Package condition defines the safety boundary for photo/text condition analysis.
package condition

import (
	"context"
	"fmt"
	"strings"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

type Rating string

const (
	VeryGood Rating = "very_good"
	Good     Rating = "good"
	Okay     Rating = "okay"
	Poor     Rating = "poor"
	Unknown  Rating = "unknown"
)

type Assessment struct {
	Rating             Rating   `json:"rating"`
	Score              float64  `json:"score"`
	Confidence         float64  `json:"confidence"`
	Signals            []string `json:"signals"`
	CleaningDifficulty string   `json:"cleaning_difficulty"`
	Provider           string   `json:"provider"`
	ProviderVersion    string   `json:"provider_version"`
}

type Input struct {
	Item          models.Item
	ImageURLs     []string
	MinConfidence float64
}

// Provider is implemented by a vision-capable adapter. Keeping it here makes
// the business rules testable without binding them to one AI vendor.
type Provider interface {
	Analyze(context.Context, Input) (Assessment, error)
}

type Service struct {
	Provider      Provider
	MinConfidence float64
}

func (s Service) Analyze(ctx context.Context, item models.Item) Assessment {
	minConfidence := s.MinConfidence
	if minConfidence <= 0 {
		minConfidence = 0.75
	}
	if s.Provider == nil {
		return unknown("no condition provider configured")
	}
	images := item.ImageURLs
	if len(images) == 0 && item.ImageURL != "" {
		images = []string{item.ImageURL}
	}
	if len(images) == 0 {
		return unknown("listing has no analysable images")
	}
	assessment, err := s.Provider.Analyze(ctx, Input{Item: item, ImageURLs: images, MinConfidence: minConfidence})
	if err != nil {
		return unknown(fmt.Sprintf("condition provider failed: %v", err))
	}
	if assessment.Confidence < minConfidence {
		assessment.Rating = Unknown
		assessment.Signals = append(assessment.Signals, "confidence below configured threshold")
	}
	if assessment.Rating == "" {
		assessment.Rating = Unknown
	}
	return assessment
}

type SafeUnknownProvider struct{}

func (SafeUnknownProvider) Analyze(context.Context, Input) (Assessment, error) {
	return unknown("condition analysis is not configured"), nil
}

func unknown(reason string) Assessment {
	return Assessment{Rating: Unknown, Signals: []string{strings.TrimSpace(reason)}, Provider: "none", ProviderVersion: "unconfigured"}
}
