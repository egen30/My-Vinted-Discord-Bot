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

	"github.com/2spy/vinted-discord-bot/internal/historysync"
	"github.com/2spy/vinted-discord-bot/internal/notify"
	"github.com/2spy/vinted-discord-bot/internal/scheduler"
	"github.com/2spy/vinted-discord-bot/internal/scrapers/vinted"
	"github.com/2spy/vinted-discord-bot/internal/store"
	"github.com/2spy/vinted-discord-bot/pkg/evaluation"
	"github.com/2spy/vinted-discord-bot/pkg/history"
	"github.com/2spy/vinted-discord-bot/pkg/logger"
	"github.com/2spy/vinted-discord-bot/pkg/models"
	"github.com/2spy/vinted-discord-bot/pkg/pricing"
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
	fallbackRules, err := pricing.ParseFallbackRules(config.resaleRules)
	if err != nil {
		logger.Error("Invalid RESALE_RULES configuration", zap.Error(err))
		os.Exit(1)
	}
	var historySyncer *historysync.Syncer
	if config.googleCredentials != "" || config.googleSheetID != "" || config.googleWorksheet != "" {
		if config.googleCredentials == "" || config.googleSheetID == "" || config.googleWorksheet == "" {
			logger.Error("Google Sheets configuration is incomplete; using fallback pricing")
		} else {
			source, sourceErr := historysync.NewGoogleSheetsSource(context.Background(), []byte(config.googleCredentials), config.googleSheetID, config.googleWorksheet)
			if sourceErr != nil {
				logger.Error("Could not configure Google Sheets sync", zap.Error(sourceErr))
			} else {
				historySyncer = historysync.New(source)
			}
		}
	}
	var salesHistory []history.Sale
	if historySyncer != nil {
		if snapshot, syncErr := historySyncer.Sync(context.Background()); syncErr != nil {
			logger.Error("Initial Google Sheets sync failed; using fallback pricing", zap.Error(syncErr))
		} else {
			salesHistory = snapshot.Sales
			logger.Info("Google Sheets history synchronized", zap.Int("accepted_rows", len(snapshot.Sales)), zap.Int("rejected_rows", len(snapshot.Rejected)))
		}
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
	lastHistorySync := time.Time{}
	syncHistory := func() {
		if historySyncer == nil || (!lastHistorySync.IsZero() && time.Since(lastHistorySync) < config.historySyncInterval) {
			return
		}
		snapshot, syncErr := historySyncer.Sync(context.Background())
		lastHistorySync = time.Now()
		if syncErr != nil {
			logger.Error("Google Sheets sync failed; retaining last good pricing snapshot", zap.Error(syncErr))
			return
		}
		salesHistory = snapshot.Sales
		if listingStore != nil {
			if persistErr := listingStore.ReplaceSalesHistory(context.Background(), salesHistory); persistErr != nil {
				logger.Error("Could not publish Google Sheets history to PostgreSQL", zap.Error(persistErr))
			}
		}
		logger.Info("Google Sheets history synchronized", zap.Int("accepted_rows", len(snapshot.Sales)), zap.Int("rejected_rows", len(snapshot.Rejected)))
	}

	processItems := func(items []models.Item) {
		for _, item := range items {
			if item.Price < float64(config.minPrice) || item.Price > float64(config.maxPrice) {
				continue
			}
			condition := evaluation.Condition(strings.ToLower(strings.TrimSpace(item.Condition)))
			model, modelErr := evaluation.MatchModel(item.Brand + " " + item.Title)
			result := evaluation.Result{Reason: "model did not match"}
			if modelErr == nil {
				estimate, hasEstimate := pricing.Estimator{Sales: salesHistory, MinimumModelData: 8, MinimumSegmentData: 5, Fallback: fallbackRules}.Estimate(model, item.Size, string(condition))
				if hasEstimate {
					result = evaluation.Evaluate(item, evaluation.PricePolicy{ExpectedResaleCents: estimate.ExpectedCents, MinimumProfitCents: config.minimumProfitCents}, condition)
				} else {
					result.Reason = "no resale estimate available"
				}
			}
			logger.Info("Listing evaluated", zap.String("item_id", item.ID), zap.String("reason", result.Reason), zap.Bool("qualified", result.Qualified))
			if config.opportunityMode == "qualified" && !result.Qualified {
				continue
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

	run := func() {
		syncHistory()
		if listingStore != nil {
			searches, err := listingStore.ListSearches(ctx, true)
			if err != nil {
				logger.Error("Could not load enabled searches", zap.Error(err))
				return
			}
			if len(searches) > 0 {
				err = scheduler.RunOnce(ctx, searches, config.searchConcurrency, func(searchCtx context.Context, search models.Search) error {
					searchJob, parseErr := vinted.JobFromSearchURL(search.URL)
					if parseErr != nil {
						_ = listingStore.RecordSearchAttempt(context.Background(), search.ID, parseErr)
						return parseErr
					}
					searchJob.SearchName = search.Name
					items, searchErr := scraper.Search(searchCtx, searchJob)
					if searchErr != nil {
						_ = listingStore.RecordSearchAttempt(context.Background(), search.ID, searchErr)
						return searchErr
					}
					_ = listingStore.RecordSearchAttempt(context.Background(), search.ID, nil)
					processItems(items)
					return nil
				})
				if err != nil {
					logger.Error("One or more Vinted searches failed", zap.Error(err))
				}
				return
			}
		}
		items, err := scraper.Search(ctx, job)
		if err != nil {
			logger.Error("Vinted search failed", zap.Error(err))
			return
		}
		processItems(items)
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
	webhookURL          string
	searchQuery         string
	vintedBaseURL       string
	catalogIDs          []string
	sizeIDs             []string
	brandIDs            []string
	currency            string
	minPrice            int
	maxPrice            int
	interval            time.Duration
	redisAddress        string
	databaseURL         string
	searchConcurrency   int
	historySyncInterval time.Duration
	resaleRules         string
	minimumProfitCents  int64
	opportunityMode     string
	googleCredentials   string
	googleSheetID       string
	googleWorksheet     string
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
		webhookURL:          strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")),
		searchQuery:         strings.TrimSpace(os.Getenv("SEARCH_QUERY")),
		vintedBaseURL:       strings.TrimSpace(os.Getenv("VINTED_BASE_URL")),
		catalogIDs:          splitIDs(os.Getenv("CATALOG_IDS")),
		sizeIDs:             splitIDs(os.Getenv("SIZE_IDS")),
		brandIDs:            splitIDs(os.Getenv("BRAND_IDS")),
		currency:            strings.TrimSpace(os.Getenv("CURRENCY")),
		minPrice:            minPrice,
		maxPrice:            maxPrice,
		interval:            time.Duration(rateMS) * time.Millisecond,
		redisAddress:        redisAddress,
		databaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		searchConcurrency:   parsePositiveEnv("SEARCH_CONCURRENCY", 2),
		resaleRules:         strings.TrimSpace(os.Getenv("RESALE_RULES")),
		minimumProfitCents:  int64(parsePositiveEnv("MIN_PROFIT_EUR", 13)) * 100,
		opportunityMode:     configuredOpportunityMode(),
		googleCredentials:   strings.TrimSpace(os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")),
		googleSheetID:       strings.TrimSpace(os.Getenv("GOOGLE_SHEET_ID")),
		googleWorksheet:     strings.TrimSpace(os.Getenv("GOOGLE_WORKSHEET")),
		historySyncInterval: time.Duration(parsePositiveEnv("HISTORY_SYNC_INTERVAL_MIN", 60)) * time.Minute,
	}, nil
}

func configuredOpportunityMode() string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OPPORTUNITY_MODE")), "qualified") {
		return "qualified"
	}
	return "shadow"
}

func parsePositiveEnv(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
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
