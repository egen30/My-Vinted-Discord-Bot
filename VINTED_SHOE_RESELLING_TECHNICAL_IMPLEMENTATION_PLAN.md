# Vinted Shoe Reselling Sourcing Assistant — Technical Implementation Plan

## Feasibility assessment

**Verdict: achievable as a phased assisted-sourcing MVP, but not with the current implementation alone.** The repository already has a useful discovery foundation: it polls Vinted search results, filters by configured IDs and price, deduplicates listing IDs in Redis, and posts a basic Discord webhook embed. That is sufficient to validate reliable new-listing detection and alert delivery.

The PRD's core decision engine still needs to be built. In particular, the current code does not identify the target model, retain listing/business state, import sales history, estimate resale value, account for costs, assess condition, calculate profit/ROI, or record reseller outcomes. A Discord incoming webhook also cannot receive `Interested`, `Purchased`, or `Ignore` interactions; those require a Discord bot/application interaction endpoint, or a separate authenticated UI/API.

The MVP is feasible if the following inputs and constraints are resolved before implementation:

- A clean export of historical purchases and sales, including at least model, EU size, purchase price, sale price, dates, and costs. Without it, initial resale estimates must be conservative, manually configured rules.
- A permitted and reliable Vinted data-access route. The current `/api/v2x/catalog/items` integration is undocumented and can change or reject requests. Treat it as a monitored dependency, respect Vinted's terms and rate limits, and do not build features intended to bypass marketplace protections.
- A vision-capable analysis provider and budget, plus a human-review policy for low-confidence or ambiguous photos. Image analysis can support decisions; it cannot make authenticity or condition guarantees.
- Confirmation of which costs are included in the €13 net-profit threshold (purchase price only, buyer protection, shipping, cleaning, selling fees, packaging, etc.).

## Current-state coverage

- **Vinted listing discovery — partial.** One configured search and catalog API polling exist; add multiple model searches, API-error handling, listing-detail enrichment, and observability.
- **Nike / EU 41–45 filtering — partial.** Existing Vinted IDs are configurable; add canonical model matching and explicit EU-size parsing/validation.
- **Duplicate prevention — partial.** A Redis key has a 30-day TTL; add persistent listing lifecycle and evaluation history.
- **Discord listing alert — partial.** The alert contains one thumbnail, price, and link; add financial fields, condition summary, image gallery, and delivery audit.
- **Historical resale estimates — missing.** Add sales-data import, robust estimator, and fallback rules.
- **Condition assessment — missing.** Add listing text/photo pipeline, vision provider, confidence threshold, and human-review policy.
- **Qualification / €13 profit — missing.** Add a cost model, profitability service, and explainable decision record.
- **Interested / purchased / sold tracking — missing.** Add database-backed state and Discord bot interactions or API.
- **Analytics and learning — missing.** Add outcome capture, aggregate queries, and prediction-accuracy reporting.

## Recommended architecture

Use a durable relational database for business records and retain Redis only for short-lived coordination/caching. PostgreSQL is the recommended production store because the system needs transactional state transitions, historical analysis, and auditable predictions. Keep the existing Go worker and Docker Compose setup; add a small Go API service when interactive actions are introduced.

```text
Vinted search/detail source
          |
          v
  discovery worker ---> PostgreSQL listings/evaluations ----> Discord notifier
          |                       |                                  |
          |                       v                                  v
          |               pricing + condition services         Discord bot actions
          |                       |                                  |
          +---- Redis cache/locks +------------------------------> PostgreSQL outcomes
                                                                    |
                                                                    v
                                                           analytics / estimator refresh
```

### Service boundaries

- **Discovery worker:** runs one search per target model/filter, normalizes items, stores newly observed listings, and requests detail enrichment where available.
- **Listing enrichment:** maps API/detail data into a stable internal record: title, description, brand, size, condition label, seller data, price breakdown, timestamps, all image URLs, and source payload version. It must handle missing fields without failing the poll.
- **Model matcher:** deterministic alias/regex rules for the five initial model groups. It returns `matched`, `ambiguous`, or `unmatched`; ambiguous listings are not automatically qualified.
- **Pricing service:** produces an explainable expected resale price from historical sales when sample-size and quality thresholds are met; otherwise uses versioned, manually maintained fallback price rules.
- **Condition service:** analyses listing text and images with a vision-capable provider, returning a rating, signals, cleaning difficulty, confidence, and a provider/model version. Low-confidence results must be marked `Unknown` rather than treated as good condition.
- **Opportunity evaluator:** combines model, size, condition, pricing, and cost rules. It stores every decision and alerts only qualified listings.
- **Discord delivery and action service:** sends rich embeds. For interactive state changes, use a Discord application/bot with signed interaction handling; a webhook alone remains suitable for one-way alerts.

## Data model

Run schema migrations from the application (for example, `golang-migrate`) and use decimal/money types rather than `float64` for all currency values.

- `target_models`: canonical name, enabled flag, aliases, allowed size range, fallback-price-rule version.
- `cost_policies`: effective date, fixed/percentage costs, included fee categories, target net profit (€13 initially).
- `listings`: platform and external ID (unique together), URL, raw title/description, brand, normalized model/size, seller metadata, item/fee/total prices, timestamps, raw payload reference, first/last seen timestamps.
- `listing_images`: listing ID, sort order, original URL, fetch/analysis status, checksum where permitted.
- `condition_assessments`: listing ID, rating, numeric score, confidence, signals JSON, cleaning difficulty, provider/model/version, created time.
- `sales_history`: imported or manually entered completed inventory record, model, size, condition, purchase/sale amounts, costs, purchase/sold dates, source and validation status.
- `resale_estimates`: model/size/condition segment, expected sale amount, low/high range, sample size, method, estimator version, generated time.
- `evaluations`: immutable snapshot of model match, condition assessment, estimate, cost policy, expected profit, maximum purchase price, ROI, confidence, qualification result, and reasons.
- `listing_status_events`: append-only lifecycle events (`new`, `evaluated`, `alerted`, `interested`, `purchased`, `sold`, `ignored`) plus actor and timestamp.
- `notification_deliveries`: evaluation, Discord message ID/channel, delivery time, and failure status for retry/audit.

## Evaluation rules

All economic values must be calculated from one stored evaluation snapshot:

```text
expected_profit = expected_resale - purchase_price - applicable_costs
maximum_purchase_price = expected_resale - applicable_costs - minimum_target_profit
roi = expected_profit / (purchase_price + applicable_costs) * 100
```

- Treat `expected_profit >= €13.00` as qualifying; equality qualifies.
- Do not impose a global `MAX_PRICE` once model-specific estimates are live. Use the calculated maximum purchase price for each listing, allowing higher-value exceptions.
- Start with conservative robust pricing: group comparable completed sales by model, then size and condition only when each segment has enough observations. Use median or trimmed/winsorized values and expose sample size/range in the evaluation.
- Define minimum sample sizes in configuration (for example, 8 model-level sales and 5 segment-level sales). Fall back to a reviewed model rule when the sample is insufficient; do not infer a high resale price from one exceptional sale.
- Exclude `Poor`, `Unknown`, and below-threshold-confidence assessments from automatic qualification. Keep them recorded for review and later calibration.
- Make every rule and model versioned so a past alert can be explained and compared with its outcome.

## Delivery phases

### Phase 0 — validate inputs and the existing foundation

1. Confirm Vinted search reliability with a manual smoke test and capture representative, sanitized fixture payloads. Add explicit non-2xx response handling, timeouts, bounded retries/backoff, and health metrics.
2. Correct configuration hygiene: remove the stray leading `2` from `.env.example`, make `MAX_PRICE` an optional temporary discovery ceiling, and document valid Nike/catalog/size IDs.
3. Add unit tests around query construction, Vinted response normalization, Redis claim/release behavior, and Discord payload generation. CI must run `go test ./...` inside the Go/Docker build environment; the current host has no `go` executable.
4. Decide and document the historical-data import format and the complete cost policy before any profitability alerts are enabled.

**Exit criteria:** a permitted source reliably returns fixtures/live results, duplicate deliveries are prevented across restarts, and basic alerts can be observed end-to-end.

### Phase 1 — deterministic profitability MVP

1. Add PostgreSQL to Compose and create the migrations/data-access layer.
2. Extend the Vinted model and normalizer to collect size, brand, description, seller data, all photos, total-price components, and update time. Add a detail fetch only if the search response lacks required evaluation fields.
3. Implement the five target-model alias rules and EU 41–45 normalizer. Store unmatched/ambiguous records but do not alert them as profitable opportunities.
4. Build CSV import with validation, duplicate detection, import report, and a manual correction path for sales history.
5. Implement versioned fallback resale rules and the cost-policy/profit calculator. Persist all evaluations, including rejected ones and their reasons.
6. Upgrade the Discord embed with model, size, price, expected resale range, expected profit, maximum price, ROI, confidence, direct link, and a clear "estimate, not guarantee" label. Use the primary image in the embed and include all original images as links or follow-up messages within Discord limits.

**Exit criteria:** the worker alerts only listings which deterministically meet target model/size and `>= €13` expected net profit using a recorded estimate and cost policy.

### Phase 2 — condition-assisted evaluation

1. Create a provider interface for image/text condition analysis and a test fake; do not couple business logic to one AI vendor.
2. Submit only the original listing images and relevant text. Store a structured response: rating, score, confidence, positive/negative observations, cleaning difficulty, and provider version.
3. Enforce strict failure behavior: unavailable provider, insufficient images, or below-confidence result becomes `Unknown` and is excluded from automatic qualification.
4. Add human review samples and measure agreement with final purchase/sale outcomes. Tune the condition adjustments only after enough labeled outcomes exist.

**Exit criteria:** an alert includes a concise, uncertainty-aware condition summary; each conclusion remains traceable to listing images and can be overridden by the reseller.

### Phase 3 — actions, outcomes, and learning

1. Introduce a Discord bot/application and a small authenticated API to process `Interested`, `Purchased`, `Sold`, and `Ignore` actions. Verify Discord request signatures and make transitions idempotent.
2. Capture actual purchase price, sale price, sale date, costs, final condition, and notes. Validate allowed state transitions in the service layer.
3. Recompute resale estimates on a scheduled job after reviewed outcomes are added. Compare predicted vs. actual resale/profit and retain the original prediction snapshot.
4. Add API/export endpoints for the PRD metrics: funnel counts, profit, median/average ROI, days to sell, results by model/size/condition, and prediction error. A dashboard is deliberately deferred.

**Exit criteria:** a reseller can update lifecycle state from Discord or API, completed outcomes are retained, and the next estimator version has measurable accuracy reporting.

## Operational and quality requirements

- Never automatically buy, message sellers, negotiate, list, or pay. The last purchase decision remains human-only.
- Protect Discord tokens, database credentials, and AI-provider keys via environment/secret management; keep `.env` untracked and rotate any exposed webhook.
- Add structured logs and metrics for poll duration, source status/rate limiting, listings discovered/evaluated/qualified, AI failures, duplicate suppression, and Discord delivery failures. Alert on sustained source or delivery failures.
- Use idempotency keys and database unique constraints so retries cannot create duplicate listings, alerts, or status events.
- Retain only the data/images necessary for the assistant; define retention/deletion policy before storing downloaded images or seller data.
- Add contract fixtures for Vinted payload changes and an alert-safe mode that persists/evaluates listings but suppresses Discord notifications during rollout.

## Rollout and acceptance plan

1. Run Phase 1 in **shadow mode** for two weeks: calculate and store evaluations, but send either no alerts or a separate review-channel alert. Compare recommendations with manual sourcing decisions.
2. Review every false positive/negative and calibrate aliases, fallback prices, and cost inclusions before enabling the main channel.
3. Enable live alerts with a conservative confidence threshold; monitor delivery rate, qualification rate, and reseller feedback weekly.
4. Only promote condition AI and automated price estimates after prediction accuracy is measured against recorded outcomes.

The 50-sales-per-month goal is a business outcome, not a software guarantee. The implementation should measure conversion from detected opportunity to purchased and sold inventory so the system can demonstrate whether it is contributing toward that goal.

## Decisions required before Phase 1

1. Provide the historical sales/purchase export and confirm the authoritative columns and currency.
2. Define every cost included in net profit and whether shipping/buyer fees vary by listing.
3. Choose the allowed Vinted data-access method and acceptable polling volume.
4. Choose the condition-analysis provider, budget, and data-retention policy.
5. Decide whether Discord actions are required in the first release; if yes, authorize creation/configuration of a Discord application/bot.
