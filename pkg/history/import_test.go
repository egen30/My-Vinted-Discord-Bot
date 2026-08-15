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
