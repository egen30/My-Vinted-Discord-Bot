package models

type ScrapeJob struct {
	ID          string   `json:"id"`
	Query       string   `json:"query"`
	Domain      string   `json:"domain"`
	CatalogIDs  []string `json:"catalog_ids"`
	SizeIDs     []string `json:"size_ids"`
	BrandIDs    []string `json:"brand_ids"`
	Currency    string   `json:"currency"`
	MinPrice    int      `json:"min_price"`
	MaxPrice    int      `json:"max_price"`
	RateLimitMs int      `json:"rate_limit_ms"`
}
