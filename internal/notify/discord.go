// Package notify delivers item alerts to external services.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

// DiscordNotifier sends listing alerts to a Discord incoming webhook.
type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewDiscordNotifier(webhookURL string) (*DiscordNotifier, error) {
	if !strings.HasPrefix(webhookURL, "https://discord.com/api/webhooks/") && !strings.HasPrefix(webhookURL, "https://discordapp.com/api/webhooks/") {
		return nil, fmt.Errorf("DISCORD_WEBHOOK_URL must be a Discord webhook URL")
	}

	return &DiscordNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// SendListing posts one listing and any available optional enrichment as a Discord embed.
func (n *DiscordNotifier) SendListing(ctx context.Context, opportunity models.Opportunity) error {
	item := opportunity.Item
	title := strings.TrimSpace(item.Title)
	if flag := countryFlag(item.Seller.Country); flag != "" {
		title = flag + " " + title
	}
	description := ""
	if item.Seller.Username != "" {
		description = fmt.Sprintf("👤 **%s**\n\n", item.Seller.Username)
	}
	if item.Description != "" {
		description += excerpt(item.Description, 280) + "\n\n"
	}
	description += fmt.Sprintf("[See more on Vinted…](%s)", item.URL)
	fields := []discordField{}
	if item.UpdatedAt != nil {
		fields = append(fields, discordField{Name: "📅 Updated", Value: relativeTime(*item.UpdatedAt), Inline: false})
	}
	if item.Size != "" {
		fields = append(fields, discordField{Name: "📏 Size", Value: item.Size, Inline: false})
	}
	if item.Brand != "" {
		fields = append(fields, discordField{Name: "🏷️ Brand", Value: item.Brand, Inline: false})
	}
	condition := item.Condition
	if condition == "" {
		condition = opportunity.Condition
	}
	if condition != "" {
		fields = append(fields, discordField{Name: "📦 Condition", Value: condition, Inline: false})
	}
	if item.Seller.Rating > 0 {
		value := stars(item.Seller.Rating)
		if item.Seller.ReviewCount > 0 {
			value += fmt.Sprintf(" (%d)", item.Seller.ReviewCount)
		}
		fields = append(fields, discordField{Name: "🌟 Rating", Value: value, Inline: false})
	}
	price := displayMoney(item.Price, item.Currency)
	if opportunity.ExpectedResaleCents > 0 {
		price += " (≈ " + displayCents(opportunity.ExpectedResaleCents, item.Currency) + ")"
	}
	fields = append(fields, discordField{Name: "💰 Price", Value: price, Inline: false})
	embed := discordEmbed{Title: title, URL: item.URL, Description: description, Color: 0x09B1BA, Fields: fields}
	imageURLs := listingImageURLs(item)
	payload := discordPayload{Embeds: []discordEmbed{embed}}
	if len(imageURLs) > 0 {
		if files, downloadErr := downloadImages(ctx, imageURLs, n.client); downloadErr == nil {
			return n.postMultipart(ctx, payload, files)
		}
	}
	if len(imageURLs) > 0 {
		payload.Embeds[0].Image.URL = imageURLs[0]
	}
	return n.postJSON(ctx, payload)
}

func (n *DiscordNotifier) postJSON(ctx context.Context, payload discordPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal Discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Discord webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return n.do(req)
}

func (n *DiscordNotifier) postMultipart(ctx context.Context, payload discordPayload, files []discordFile) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	payload.Attachments = make([]discordAttachment, 0, len(files))
	for i, file := range files {
		payload.Attachments = append(payload.Attachments, discordAttachment{ID: i, Filename: file.Name})
		filePart, fileErr := writer.CreateFormFile(fmt.Sprintf("files[%d]", i), file.Name)
		if fileErr != nil {
			return fmt.Errorf("create Discord image part: %w", fileErr)
		}
		if _, fileErr = filePart.Write(file.Data); fileErr != nil {
			return fmt.Errorf("write Discord image part: %w", fileErr)
		}
	}
	payloadPart, err := writer.CreateFormField("payload_json")
	if err != nil {
		return fmt.Errorf("create Discord payload part: %w", err)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal Discord payload: %w", err)
	}
	if _, err := payloadPart.Write(encoded); err != nil {
		return fmt.Errorf("write Discord payload part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close Discord multipart body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, &body)
	if err != nil {
		return fmt.Errorf("create Discord webhook request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return n.do(req)
}

func (n *DiscordNotifier) do(req *http.Request) error {

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send Discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("Discord webhook returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func countryFlag(country string) string {
	country = strings.TrimSpace(country)
	if len([]rune(country)) == 2 {
		letters := []rune(strings.ToUpper(country))
		if letters[0] >= 'A' && letters[0] <= 'Z' && letters[1] >= 'A' && letters[1] <= 'Z' {
			return string([]rune{0x1F1E6 + letters[0] - 'A', 0x1F1E6 + letters[1] - 'A'})
		}
	}
	switch strings.ToLower(country) {
	case "france":
		return "🇫🇷"
	case "germany", "deutschland":
		return "🇩🇪"
	case "netherlands", "the netherlands":
		return "🇳🇱"
	case "belgium":
		return "🇧🇪"
	default:
		return ""
	}
}

func relativeTime(value time.Time) string {
	age := time.Since(value)
	if age < 0 {
		return "just now"
	}
	switch {
	case age < time.Minute:
		return "just now"
	case age < time.Hour:
		minutes := int(age / time.Minute)
		return fmt.Sprintf("%d minute%s ago", minutes, pluralSuffix(minutes))
	case age < 24*time.Hour:
		hours := int(age / time.Hour)
		return fmt.Sprintf("%d hour%s ago", hours, pluralSuffix(hours))
	default:
		days := int(age / (24 * time.Hour))
		return fmt.Sprintf("%d day%s ago", days, pluralSuffix(days))
	}
}

func pluralSuffix(value int) string {
	if value == 1 {
		return ""
	}
	return "s"
}

func stars(rating float64) string {
	count := int(rating + 0.5)
	if count < 1 {
		count = 1
	}
	if count > 5 {
		count = 5
	}
	return strings.Repeat("⭐️", count)
}

func displayCents(cents int64, currency string) string {
	return displayMoney(float64(cents)/100, currency)
}

func displayMoney(amount float64, currency string) string {
	symbol := map[string]string{"EUR": "€", "GBP": "£", "USD": "$"}[strings.ToUpper(strings.TrimSpace(currency))]
	if symbol == "" {
		symbol = strings.TrimSpace(currency)
	}
	return fmt.Sprintf("%.2f %s", amount, symbol)
}

func listingImageURLs(item models.Item) []string {
	urls := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	for _, imageURL := range append([]string{item.ImageURL}, item.ImageURLs...) {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL == "" {
			continue
		}
		if _, ok := seen[imageURL]; ok {
			continue
		}
		seen[imageURL] = struct{}{}
		urls = append(urls, imageURL)
		if len(urls) == 3 {
			break
		}
	}
	return urls
}

type discordFile struct {
	Name string
	Data []byte
}

func downloadImages(ctx context.Context, imageURLs []string, client *http.Client) ([]discordFile, error) {
	files := make([]discordFile, 0, len(imageURLs))
	for i, imageURL := range imageURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create image request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download image: %w", err)
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			resp.Body.Close()
			return nil, fmt.Errorf("image returned %s", resp.Status)
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read image: %w", readErr)
		}
		files = append(files, discordFile{Name: fmt.Sprintf("vinted-photo-%d.jpg", i+1), Data: data})
	}
	return files, nil
}

type discordPayload struct {
	Embeds      []discordEmbed      `json:"embeds"`
	Attachments []discordAttachment `json:"attachments,omitempty"`
}

type discordAttachment struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
}

type discordEmbed struct {
	Title       string         `json:"title,omitempty"`
	URL         string         `json:"url,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color,omitempty"`
	Fields      []discordField `json:"fields,omitempty"`
	Image       discordImage   `json:"image,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordImage struct {
	URL string `json:"url,omitempty"`
}

func excerpt(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength-1] + "…"
}
