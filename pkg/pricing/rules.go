package pricing

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseFallbackRules parses `model=amount,model=amount` in the configured currency.
func ParseFallbackRules(value string) (map[string]Estimate, error) {
	rules := make(map[string]Estimate)
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid resale rule %q", entry)
		}
		amount, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || amount <= 0 {
			return nil, fmt.Errorf("invalid resale amount in %q", entry)
		}
		cents := int64(amount*100 + 0.5)
		rules[strings.ToLower(strings.TrimSpace(parts[0]))] = Estimate{ExpectedCents: cents, LowCents: cents, HighCents: cents, Method: "manual_fallback"}
	}
	return rules, nil
}
