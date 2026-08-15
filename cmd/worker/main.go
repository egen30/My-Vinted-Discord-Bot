package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/2spy/vinted-discord-bot/internal/notify"
	"github.com/2spy/vinted-discord-bot/internal/scrapers/vinted"
	"github.com/2spy/vinted-discord-bot/internal/store"
	"github.com/2spy/vinted-discord-bot/pkg/logger"
	"github.com/2spy/vinted-discord-bot/pkg/models"
	"go.uber.org/zap"
)

const deliveredItemTTL = 30 * 24 * time.Hour
const notificationDelay = time.Second

func main() {
	logger.Init()
	defer logger.Logger.Sync()

	config, err := loadConfig()
	if err != nil {
		logger.Error("Invalid worker configuration", zap.Error(err))
		os.Exit(1)
	}

	notifier, err := notify.NewDiscordNotifier(config.webhookURL)
	if err != nil {
		logger.Error("Invalid Discord configuration", zap.Error(err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	scraper := vinted.NewVintedScraper()
	deduplicator := store.NewRedisDeduplicator(config.redisAddress)
	var listingStore *store.PostgresStore
	if config.databaseURL != "" {
		listingStore, err = store.NewPostgresStore(ctx, config.databaseURL)
		if err != nil {
			logger.Error("PostgreSQL unavailable", zap.Error(err))
			os.Exit(1)
		}
		defer listingStore.Close()
	}
	job := models.ScrapeJob{
		Query:      config.searchQuery,
		Domain:     config.vintedBaseURL,
		CatalogIDs: config.catalogIDs,
		SizeIDs:    config.sizeIDs,
		BrandIDs:   config.brandIDs,
		Currency:   config.currency,
		MinPrice:   config.minPrice,
		MaxPrice:   config.maxPrice,
	}

	run := func() {
		items, err := scraper.Search(ctx, job)
		if err != nil {
			logger.Error("Vinted search failed", zap.Error(err))
			return
		}

		for _, item := range items {
			if item.Price < float64(config.minPrice) || item.Price > float64(config.maxPrice) {
				continue
			}
			if listingStore != nil {
				if _, err := listingStore.UpsertListing(ctx, item); err != nil {
					logger.Error("Could not persist listing", zap.String("item_id", item.ID), zap.Error(err))
					continue
				}
			}

			// Claim the item before delivering so a later poll cannot send it twice.
			// Release the claim if Discord rejects the message, allowing a retry.
			isNew, err := deduplicator.MarkDelivered(ctx, item.ID, deliveredItemTTL)
			if err != nil {
				logger.Error("Redis deduplication failed", zap.String("item_id", item.ID), zap.Error(err))
				continue
			}
			if !isNew {
				continue
			}

			if err := notifier.SendItem(ctx, item); err != nil {
				logger.Error("Discord notification failed", zap.String("item_id", item.ID), zap.Error(err))
				if forgetErr := deduplicator.Forget(ctx, item.ID); forgetErr != nil {
					logger.Error("Could not release failed notification", zap.String("item_id", item.ID), zap.Error(forgetErr))
				}
				if !waitForNextNotification(ctx) {
					return
				}
				continue
			}
			logger.Info("Discord notification sent", zap.String("item_id", item.ID), zap.String("title", item.Title))
			if !waitForNextNotification(ctx) {
				return
			}
		}
	}

	logger.Info("Worker started", zap.String("query", config.searchQuery), zap.Int("max_price", config.maxPrice), zap.Duration("interval", config.interval))
	run()
	ticker := time.NewTicker(config.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker stopped")
			return
		case <-ticker.C:
			run()
		}
	}
}

type config struct {
	webhookURL    string
	searchQuery   string
	vintedBaseURL string
	catalogIDs    []string
	sizeIDs       []string
	brandIDs      []string
	currency      string
	minPrice      int
	maxPrice      int
	interval      time.Duration
	redisAddress  string
	databaseURL   string
}

func loadConfig() (config, error) {
	maxPrice, err := strconv.Atoi(os.Getenv("MAX_PRICE"))
	if err != nil || maxPrice <= 0 {
		return config{}, fmt.Errorf("MAX_PRICE must be a positive whole number")
	}
	rateMS := 15000
	if rawRate := os.Getenv("RATE_LIMIT_MS"); rawRate != "" {
		rateMS, err = strconv.Atoi(rawRate)
		if err != nil || rateMS < 1000 {
			return config{}, fmt.Errorf("RATE_LIMIT_MS must be at least 1000")
		}
	}
	minPrice := 0
	if rawMinPrice := os.Getenv("MIN_PRICE"); rawMinPrice != "" {
		minPrice, err = strconv.Atoi(rawMinPrice)
		if err != nil || minPrice < 0 || minPrice > maxPrice {
			return config{}, fmt.Errorf("MIN_PRICE must be between 0 and MAX_PRICE")
		}
	}
	redisAddress := os.Getenv("REDIS_ADDR")
	if redisAddress == "" {
		redisAddress = "localhost:6379"
	}
	return config{
		webhookURL:    strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")),
		searchQuery:   strings.TrimSpace(os.Getenv("SEARCH_QUERY")),
		vintedBaseURL: strings.TrimSpace(os.Getenv("VINTED_BASE_URL")),
		catalogIDs:    splitIDs(os.Getenv("CATALOG_IDS")),
		sizeIDs:       splitIDs(os.Getenv("SIZE_IDS")),
		brandIDs:      splitIDs(os.Getenv("BRAND_IDS")),
		currency:      strings.TrimSpace(os.Getenv("CURRENCY")),
		minPrice:      minPrice,
		maxPrice:      maxPrice,
		interval:      time.Duration(rateMS) * time.Millisecond,
		redisAddress:  redisAddress,
		databaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
	}, nil
}

func waitForNextNotification(ctx context.Context) bool {
	timer := time.NewTimer(notificationDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func splitIDs(value string) []string {
	var ids []string
	for _, id := range strings.Split(value, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
