package models

import "time"

// ListingSummary is the read model used by the admin page. It deliberately
// contains only fields needed for operations, not the raw marketplace payload.
type ListingSummary struct {
	ID          int64     `json:"id"`
	ExternalID  string    `json:"external_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	PriceCents  int64     `json:"price_cents"`
	Currency    string    `json:"currency"`
	Seller      string    `json:"seller,omitempty"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	SearchNames []string  `json:"search_names"`
}
