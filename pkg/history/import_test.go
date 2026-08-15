package history

import (
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	sales, err := ParseCSV(strings.NewReader("model,size,condition,purchase_price,sale_price,costs,sold_at\nNike P-6000,42,good,15.00,35,2.50,2026-01-02\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sales) != 1 || sales[0].PurchaseCents != 1500 || sales[0].SaleCents != 3500 || sales[0].CostsCents != 250 {
		t.Fatalf("unexpected import: %+v", sales)
	}
}

func TestParseCSVMissingColumn(t *testing.T) {
	if _, err := ParseCSV(strings.NewReader("model,purchase_price\nNike P-6000,15\n")); err == nil {
		t.Fatal("expected missing column error")
	}
}

func TestParseCSVNormalizesProductAndPreservesSourceValues(t *testing.T) {
	sales, err := ParseCSV(strings.NewReader("model,size,condition,purchase_price,sale_price,purchased_at,sold_at\nP 6000 Nike,EU 42,Very Good,15,35,2026-01-01,2026-01-08\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sales) != 1 {
		t.Fatalf("expected one sale, got %d", len(sales))
	}
	sale := sales[0]
	if sale.Model != "P 6000 Nike" || sale.OriginalModel != "P 6000 Nike" {
		t.Fatalf("source model was not preserved: %+v", sale)
	}
	if sale.Brand != "Nike" || sale.NormalizedModel != "P-6000" || sale.NormalizedSize != "42" || sale.NormalizedCondition != "very_good" {
		t.Fatalf("unexpected normalized values: %+v", sale)
	}
	if sale.DaysToSell == nil || *sale.DaysToSell != 7 || sale.SourceRow != 2 {
		t.Fatalf("unexpected derived history fields: %+v", sale)
	}
}

func TestNormalizeSaleDoesNotInventDaysForReversedDates(t *testing.T) {
	sales, err := ParseCSV(strings.NewReader("model,purchase_price,sale_price,purchased_at,sold_at\nNike P-6000,15,35,2026-01-08,2026-01-01\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sales[0].DaysToSell != nil {
		t.Fatalf("expected unknown days to sell, got %v", *sales[0].DaysToSell)
	}
}
