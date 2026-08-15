package pricing

import "testing"

func TestParseFallbackRules(t *testing.T) {
	rules, err := ParseFallbackRules("Nike P-6000=35, Nike Shox=40")
	if err != nil || rules["nike p-6000"].ExpectedCents != 3500 || rules["nike shox"].ExpectedCents != 4000 {
		t.Fatalf("unexpected rules: %+v, %v", rules, err)
	}
}
