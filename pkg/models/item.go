package models

type Item struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Brand       string   `json:"brand"`
	Size        string   `json:"size"`
	Condition   string   `json:"condition"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Currency    string   `json:"currency"`
	URL         string   `json:"url"`
	ImageURL    string   `json:"image_url"`
	ImageURLs   []string `json:"image_urls,omitempty"`
	Platform    string   `json:"platform"`
}
