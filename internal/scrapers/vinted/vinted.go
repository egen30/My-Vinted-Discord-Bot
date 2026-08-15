package vinted

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/2spy/vinted-discord-bot/pkg/logger"
	"github.com/2spy/vinted-discord-bot/pkg/models"
	"github.com/2spy/vinted-discord-bot/pkg/stealth"
	http "github.com/bogdanfinn/fhttp"
	"go.uber.org/zap"
)

type VintedScraper struct {
	client httpClient
}

// JobFromSearchURL converts a user-managed Vinted URL into the existing API job.
func JobFromSearchURL(rawURL string) (models.ScrapeJob, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || !supportedHost(parsed.Hostname()) {
		return models.ScrapeJob{}, fmt.Errorf("unsupported Vinted search URL")
	}
	query := parsed.Query()
	job := models.ScrapeJob{
		Domain:     parsed.Scheme + "://" + parsed.Host,
		Query:      query.Get("search_text"),
		CatalogIDs: splitQueryIDs(query, "catalog_ids", "catalog[]"),
		SizeIDs:    splitQueryIDs(query, "size_ids", "size[]"),
		BrandIDs:   splitQueryIDs(query, "brand_ids", "brand_ids[]"),
		Currency:   query.Get("currency"),
	}
	if job.Currency == "" {
		job.Currency = "EUR"
	}
	if value := query.Get("price_from"); value != "" {
		job.MinPrice, err = strconv.Atoi(value)
		if err != nil || job.MinPrice < 0 {
			return models.ScrapeJob{}, fmt.Errorf("invalid price_from")
		}
	}
	if value := query.Get("price_to"); value != "" {
		job.MaxPrice, err = strconv.Atoi(value)
		if err != nil || job.MaxPrice < 0 {
			return models.ScrapeJob{}, fmt.Errorf("invalid price_to")
		}
	}
	return job, nil
}

func supportedHost(host string) bool {
	return host == "vinted.de" || strings.HasSuffix(host, ".vinted.de") || host == "vinted.fr" || strings.HasSuffix(host, ".vinted.fr")
}

func splitQueryIDs(query url.Values, names ...string) []string {
	var result []string
	for _, name := range names {
		for _, value := range query[name] {
			for _, id := range strings.Split(value, ",") {
				if id = strings.TrimSpace(id); id != "" {
					result = append(result, id)
				}
			}
		}
	}
	return result
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

const (
	maxResponseBytes   = 8 << 20
	maxRequestAttempts = 3
)

func NewVintedScraper() *VintedScraper {
	client, err := stealth.CreateClient()
	if err != nil {
		panic(err)
	}
	return &VintedScraper{
		client: client,
	}
}

func (s *VintedScraper) Search(ctx context.Context, job models.ScrapeJob) ([]models.Item, error) {
	logger.Info("Searching on Vinted", zap.String("query", job.Query))
	client := s.client
	baseURL := job.Domain
	if baseURL == "" {
		baseURL = "https://www.vinted.de"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	initialURL, err := url.Parse(baseURL + "/catalog")
	if err != nil {
		return nil, fmt.Errorf("parse Vinted base URL: %w", err)
	}
	initialQuery := initialURL.Query()
	initialQuery.Set("search_text", job.Query)
	initialURL.RawQuery = initialQuery.Encode()
	reqInit, err := http.NewRequestWithContext(ctx, "GET", initialURL.String(), nil)
	if err != nil {
		logger.Error("Error creating request", zap.Error(err))
		return nil, err
	}
	reqInit.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	reqInit.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	reqInit.Header.Set("Accept-Language", "de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7")
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	reqInit.Header.Set("Connection", "keep-alive")
	reqInit.Header.Set("Referer", baseURL+"/")
	reqInit.Header.Set("Upgrade-Insecure-Requests", "1")
	reqInit.Header.Set("Sec-Fetch-Dest", "document")
	reqInit.Header.Set("Sec-Fetch-Mode", "navigate")
	reqInit.Header.Set("Sec-Fetch-Site", "same-origin")
	reqInit.Header.Set("Sec-Fetch-User", "?1")
	reqInit.Header.Set("Priority", "u=0, i")
	reqInit.Header.Set("Pragma", "no-cache")
	reqInit.Header.Set("Cache-Control", "no-cache")
	reqInit.Header.Set("TE", "trailers")
	respInit, err := doWithRetry(ctx, client, reqInit)
	if err != nil {
		logger.Error("Error making request", zap.Error(err))
		return nil, err
	}
	_, err = readResponseBody(respInit)
	if err != nil {
		logger.Error("Error reading response body", zap.Error(err))
		return nil, err
	}
	apiURL, err := url.Parse(baseURL + "/api/v2/catalog/items")
	if err != nil {
		return nil, fmt.Errorf("parse Vinted API URL: %w", err)
	}
	apiQuery := apiURL.Query()
	apiQuery.Set("page", "1")
	apiQuery.Set("per_page", "96")
	apiQuery.Set("order", "newest_first")
	apiQuery.Set("search_text", job.Query)
	apiQuery.Set("catalog_ids", strings.Join(job.CatalogIDs, ","))
	apiQuery.Set("size_ids", strings.Join(job.SizeIDs, ","))
	apiQuery.Set("brand_ids", strings.Join(job.BrandIDs, ","))
	if job.MinPrice > 0 {
		apiQuery.Set("price_from", strconv.Itoa(job.MinPrice))
	}
	if job.MaxPrice > 0 {
		apiQuery.Set("price_to", strconv.Itoa(job.MaxPrice))
	}
	if job.Currency != "" {
		apiQuery.Set("currency", job.Currency)
	}
	apiURL.RawQuery = apiQuery.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL.String(), nil)
	if err != nil {
		logger.Error("Error making request", zap.Error(err))
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7")
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("TE", "trailers")
	resp, err := doWithRetry(ctx, client, req)
	if err != nil {
		logger.Error("Error making request", zap.Error(err))
		return nil, err
	}
	bodyText, err := readResponseBody(resp)
	if err != nil {
		logger.Error("Error reading response body", zap.Error(err))
		return nil, err
	}
	var response VintedResponse
	err = json.Unmarshal(bodyText, &response)
	if err != nil {
		logger.Error("Error unmarshalling response", zap.Error(err))
		return nil, err
	}
	var items []models.Item
	for _, item := range response.Items {
		price, err := strconv.ParseFloat(item.Price.Amount, 64)
		if err != nil {
			logger.Error("Error parsing price", zap.Error(err))
			continue
		}
		items = append(items, models.Item{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     item.Title,
			Brand:     item.BrandTitle,
			Size:      item.SizeTitle,
			Price:     price,
			Currency:  item.Price.CurrencyCode,
			URL:       item.URL,
			ImageURL:  item.Photo.URL,
			ImageURLs: imageURLs(item.Photos, item.Photo.URL),
			Platform:  "vinted",
			Seller: models.Seller{
				Username:    item.User.Login,
				ProfileURL:  item.User.ProfileURL,
				AvatarURL:   item.User.Photo.URL,
				Rating:      item.User.Feedback.Score,
				ReviewCount: item.User.Feedback.Count,
			},
		})
	}
	return items, nil
}

func imageURLs(photos []struct {
	ID                  int64  `json:"id"`
	ImageNo             int    `json:"image_no"`
	Width               int    `json:"width"`
	Height              int    `json:"height"`
	DominantColor       string `json:"dominant_color"`
	DominantColorOpaque string `json:"dominant_color_opaque"`
	URL                 string `json:"url"`
	IsMain              bool   `json:"is_main"`
	Thumbnails          []struct {
		Type         string      `json:"type"`
		URL          string      `json:"url"`
		Width        int         `json:"width"`
		Height       int         `json:"height"`
		OriginalSize interface{} `json:"original_size"`
	} `json:"thumbnails"`
	HighResolution struct {
		ID          string      `json:"id"`
		Timestamp   int         `json:"timestamp"`
		Orientation interface{} `json:"orientation"`
	} `json:"high_resolution"`
	IsSuspicious bool     `json:"is_suspicious"`
	FullSizeURL  string   `json:"full_size_url"`
	IsHidden     bool     `json:"is_hidden"`
	Extra        struct{} `json:"extra"`
}, primary string) []string {
	urls := make([]string, 0, len(photos)+1)
	seen := make(map[string]struct{})
	appendURL := func(value string) {
		if value != "" {
			if _, ok := seen[value]; !ok {
				seen[value] = struct{}{}
				urls = append(urls, value)
			}
		}
	}
	appendURL(primary)
	for _, photo := range photos {
		appendURL(photo.URL)
		appendURL(photo.FullSizeURL)
	}
	return urls
}

func doWithRetry(ctx context.Context, client httpClient, req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < maxRequestAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("Vinted request returned %s", resp.Status)
			_ = resp.Body.Close()
		}
		if attempt == maxRequestAttempts-1 {
			break
		}
		delay := time.Duration(1<<attempt) * 250 * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("Vinted request failed after %d attempts: %w", maxRequestAttempts, lastErr)
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("Vinted returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("Vinted response exceeds %d bytes", maxResponseBytes)
	}
	return body, nil
}

func (s *VintedScraper) Name() string {
	return "vinted"
}
