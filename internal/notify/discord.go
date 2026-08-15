// Package notify delivers item alerts to external services.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// SendItem posts one listing as a Discord embed.
func (n *DiscordNotifier) SendItem(ctx context.Context, item models.Item) error {
	return n.SendOpportunity(ctx, models.Opportunity{Item: item})
}

// SendOpportunity posts the financial evaluation and original listing images.
func (n *DiscordNotifier) SendOpportunity(ctx context.Context, opportunity models.Opportunity) error {
	item := opportunity.Item
	title := item.Title
	if item.Brand != "" {
		title = item.Brand + " | " + title
	}
	description := fmt.Sprintf("**Price:** %.2f %s\n[Open listing](%s)", item.Price, item.Currency, item.URL)
	fields := []discordField{}
	if opportunity.ExpectedResaleCents > 0 {
		fields = append(fields,
			discordField{Name: "Expected resale", Value: formatCents(opportunity.ExpectedResaleCents, item.Currency), Inline: true},
			discordField{Name: "Expected profit", Value: formatCents(opportunity.ExpectedProfitCents, item.Currency), Inline: true},
			discordField{Name: "Maximum purchase", Value: formatCents(opportunity.MaximumPurchaseCents, item.Currency), Inline: true},
			discordField{Name: "ROI", Value: fmt.Sprintf("%.1f%%", opportunity.ROIPercent), Inline: true},
		)
	}
	if opportunity.Condition != "" {
		fields = append(fields, discordField{Name: "Condition", Value: opportunity.Condition, Inline: true})
	}
	if item.Size != "" {
		fields = append(fields, discordField{Name: "Size", Value: item.Size, Inline: true})
	}
	embed := discordEmbed{Title: title, URL: item.URL, Description: description, Color: 0x09B1BA, Fields: fields}
	if item.ImageURL != "" {
		embed.Image.URL = item.ImageURL
	}
	payload := discordPayload{Embeds: []discordEmbed{embed}}
	for _, imageURL := range item.ImageURLs {
		if imageURL != "" && imageURL != item.ImageURL && len(payload.Embeds) < 10 {
			payload.Embeds = append(payload.Embeds, discordEmbed{Image: discordImage{URL: imageURL}})
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal Discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Discord webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

type discordPayload struct {
	Embeds []discordEmbed `json:"embeds"`
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

func formatCents(cents int64, currency string) string {
	return fmt.Sprintf("%.2f %s", float64(cents)/100, currency)
}
