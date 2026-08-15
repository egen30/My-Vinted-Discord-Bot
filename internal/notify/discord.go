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
	payload := struct {
		Embeds []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Color       int    `json:"color"`
			Thumbnail   struct {
				URL string `json:"url"`
			} `json:"thumbnail"`
		} `json:"embeds"`
	}{Embeds: make([]struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		Description string `json:"description"`
		Color       int    `json:"color"`
		Thumbnail   struct {
			URL string `json:"url"`
		} `json:"thumbnail"`
	}, 1)}

	embed := &payload.Embeds[0]
	embed.Title = item.Title
	embed.URL = item.URL
	embed.Description = fmt.Sprintf("**Price:** %.2f %s\n[Open listing](%s)", item.Price, item.Currency, item.URL)
	embed.Color = 0x09B1BA
	embed.Thumbnail.URL = item.ImageURL

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
