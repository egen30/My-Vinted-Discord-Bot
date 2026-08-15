// Package history parses the reseller's completed-sales export.
package history

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Sale struct {
	Model         string
	Size          string
	Condition     string
	PurchaseCents int64
	SaleCents     int64
	CostsCents    int64
	PurchasedAt   *time.Time
	SoldAt        *time.Time
	Source        string
}

// ParseCSV accepts the required model, purchase_price, and sale_price columns
// plus optional size, condition, costs, purchased_at, sold_at, and source.
func ParseCSV(r io.Reader) ([]Sale, error) {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read sales CSV header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for index, value := range header {
		columns[strings.ToLower(strings.TrimSpace(value))] = index
	}
	for _, required := range []string{"model", "purchase_price", "sale_price"} {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("sales CSV missing required column %q", required)
		}
	}
	var result []Sale
	for rowNumber := 2; ; rowNumber++ {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read sales CSV row %d: %w", rowNumber, readErr)
		}
		model := strings.TrimSpace(field(row, columns, "model"))
		if model == "" {
			return nil, fmt.Errorf("sales CSV row %d has empty model", rowNumber)
		}
		purchase, err := parseCents(field(row, columns, "purchase_price"))
		if err != nil {
			return nil, fmt.Errorf("row %d purchase_price: %w", rowNumber, err)
		}
		sale, err := parseCents(field(row, columns, "sale_price"))
		if err != nil {
			return nil, fmt.Errorf("row %d sale_price: %w", rowNumber, err)
		}
		costs, err := parseOptionalCents(field(row, columns, "costs"))
		if err != nil {
			return nil, fmt.Errorf("row %d costs: %w", rowNumber, err)
		}
		purchasedAt, err := parseDate(field(row, columns, "purchased_at"))
		if err != nil {
			return nil, fmt.Errorf("row %d purchased_at: %w", rowNumber, err)
		}
		soldAt, err := parseDate(field(row, columns, "sold_at"))
		if err != nil {
			return nil, fmt.Errorf("row %d sold_at: %w", rowNumber, err)
		}
		result = append(result, Sale{Model: model, Size: strings.TrimSpace(field(row, columns, "size")), Condition: strings.TrimSpace(field(row, columns, "condition")), PurchaseCents: purchase, SaleCents: sale, CostsCents: costs, PurchasedAt: purchasedAt, SoldAt: soldAt, Source: strings.TrimSpace(field(row, columns, "source"))})
	}
	return result, nil
}

func field(row []string, columns map[string]int, name string) string {
	index, ok := columns[name]
	if !ok || index >= len(row) {
		return ""
	}
	return row[index]
}

func parseOptionalCents(value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return parseCents(value)
}

func parseCents(value string) (int64, error) {
	clean := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "€"))
	clean = strings.ReplaceAll(clean, ",", ".")
	amount, err := strconv.ParseFloat(clean, 64)
	if err != nil || amount < 0 {
		return 0, fmt.Errorf("invalid non-negative amount %q", value)
	}
	return int64(amount*100 + 0.5), nil
}

func parseDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "02.01.2006"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid date %q", value)
}
