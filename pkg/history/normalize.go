package history

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var nonAlphaNumeric = regexp.MustCompile(`[^a-z0-9]+`)
var whitespace = regexp.MustCompile(`\s+`)

// NormalizeSale derives stable comparison fields while retaining the values
// entered in the source sheet. It is intentionally conservative: unknown
// products are normalized for whitespace/casing but are not guessed into a
// known catalog model.
func NormalizeSale(sale Sale) Sale {
	sale.OriginalModel = strings.TrimSpace(sale.Model)
	sale.Brand, sale.NormalizedModel = normalizeProduct(sale.OriginalModel)
	sale.NormalizedSize = normalizeSize(sale.Size)
	sale.NormalizedCondition = normalizeCondition(sale.Condition)
	if sale.PurchasedAt != nil && sale.SoldAt != nil && !sale.SoldAt.Before(*sale.PurchasedAt) {
		days := int(sale.SoldAt.Sub(*sale.PurchasedAt) / (24 * time.Hour))
		sale.DaysToSell = &days
	}
	return sale
}

func normalizeProduct(value string) (string, string) {
	normalized := strings.ToLower(whitespace.ReplaceAllString(strings.TrimSpace(value), " "))
	compact := nonAlphaNumeric.ReplaceAllString(normalized, "")
	known := map[string]struct {
		brand string
		model string
	}{
		"nikep6000":    {"Nike", "P-6000"},
		"p6000nike":    {"Nike", "P-6000"},
		"nikeairmax95": {"Nike", "Air Max 95"},
		"nikeairmax90": {"Nike", "Air Max 90"},
		"nikeairmax97": {"Nike", "Air Max 97"},
		"nikeshox":     {"Nike", "Shox"},
	}
	if product, ok := known[compact]; ok {
		return product.brand, product.model
	}
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return "", ""
	}
	brand := titleWords(parts[0])
	model := strings.TrimSpace(strings.Join(parts[1:], " "))
	if model == "" {
		model = brand
		brand = ""
	}
	return brand, titleWords(model)
}

func normalizeSize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "eu ")
	value = strings.TrimSpace(strings.TrimPrefix(value, "eur "))
	if parsed, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64); err == nil {
		return strconv.FormatFloat(parsed, 'f', -1, 64)
	}
	return strings.Join(strings.Fields(value), " ")
}

func normalizeCondition(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "new", "new with tags", "new without tags", "brand new":
		return "new"
	case "very good", "very_good", "like new", "excellent":
		return "very_good"
	case "good", "great":
		return "good"
	case "satisfactory", "okay", "ok", "fair":
		return "okay"
	case "poor", "bad":
		return "poor"
	default:
		return strings.Join(strings.Fields(strings.ToLower(value)), " ")
	}
}

func titleWords(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		if word == "p-6000" {
			words[i] = "P-6000"
			continue
		}
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}
