// Package pricing estimates resale values from completed sales.
package pricing

import (
	"sort"
	"strings"

	"github.com/2spy/vinted-discord-bot/pkg/history"
)

type Key struct {
	Model     string
	Size      string
	Condition string
}

type Estimate struct {
	ExpectedCents int64
	LowCents      int64
	HighCents     int64
	Samples       int
	Method        string
}

type Estimator struct {
	Sales              []history.Sale
	MinimumModelData   int
	MinimumSegmentData int
	Fallback           map[string]Estimate
}

func (e Estimator) Estimate(model, size, condition string) (Estimate, bool) {
	model = strings.ToLower(strings.TrimSpace(model))
	condition = strings.ToLower(strings.TrimSpace(condition))
	segment := make([]int64, 0)
	modelPrices := make([]int64, 0)
	for _, sale := range e.Sales {
		saleModel := sale.NormalizedModel
		if saleModel == "" {
			saleModel = sale.Model
		}
		if strings.ToLower(strings.TrimSpace(saleModel)) != model {
			continue
		}
		modelPrices = append(modelPrices, sale.SaleCents)
		saleSize := sale.NormalizedSize
		if saleSize == "" {
			saleSize = sale.Size
		}
		saleCondition := sale.NormalizedCondition
		if saleCondition == "" {
			saleCondition = sale.Condition
		}
		if strings.ToLower(strings.TrimSpace(saleSize)) == strings.ToLower(strings.TrimSpace(size)) && strings.ToLower(strings.TrimSpace(saleCondition)) == condition {
			segment = append(segment, sale.SaleCents)
		}
	}
	if len(segment) >= e.MinimumSegmentData {
		return robustEstimate(segment, "historical_segment"), true
	}
	if len(modelPrices) >= e.MinimumModelData {
		return robustEstimate(modelPrices, "historical_model"), true
	}
	if fallback, ok := e.Fallback[model]; ok {
		return fallback, true
	}
	return Estimate{}, false
}

func robustEstimate(values []int64, method string) Estimate {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	lowIndex := len(values) / 10
	highIndex := len(values) - 1 - lowIndex
	trimmed := values[lowIndex : highIndex+1]
	median := trimmed[len(trimmed)/2]
	if len(trimmed)%2 == 0 {
		median = (trimmed[len(trimmed)/2-1] + trimmed[len(trimmed)/2]) / 2
	}
	return Estimate{ExpectedCents: median, LowCents: trimmed[0], HighCents: trimmed[len(trimmed)-1], Samples: len(values), Method: method}
}
