# Vinted Shoe Reselling Sourcing Assistant

A Go/Docker service that monitors Vinted searches and sends new-listing alerts to Discord. It is an assistant: purchasing, seller communication, payments, and resale listings remain manual.

## What is implemented

- Named Vinted search management through an authenticated HTTP API.
- A simple admin page at `/admin` for adding, editing, enabling, disabling, and deleting searches.
- Bounded multi-search polling with a legacy environment-search fallback.
- Persistent normalized listing records, search attribution, and notification delivery history in PostgreSQL.
- Redis alert deduplication with a 30-day TTL; overlapping searches notify only once.
- Seller metadata, descriptions, original image URLs, and source-search names in Discord alerts.
- In-memory model, size, condition, resale-price, profit, and ROI evaluation.
- Optional read-only Google Sheets history synchronization with row validation and atomic snapshots.
- Discord embeds with listing links, photos, seller metadata, source searches, and optional financial fields.

## Requirements

- Docker and Docker Compose
- A Discord webhook URL
- For historical pricing: a Google service account with read-only access to the sales spreadsheet

The Vinted catalog endpoint used by this project is undocumented and may change. Use an allowed access method, respect Vinted's terms and rate limits, and do not use the bot for automated purchases or seller actions.

## Quick start

```bash
git clone https://github.com/egen30/My-Vinted-Discord-Bot.git
cd My-Vinted-Discord-Bot
cp .env.example .env
```

Edit `.env` before starting the stack. The required setup is described below.

```bash
docker compose up -d --build
docker compose ps
docker compose logs -f worker
```

PostgreSQL and Redis are started by Compose. PostgreSQL stores search configuration, normalized listings, search attribution, notification audit records, and synchronized sales history. Redis stores short-lived delivery claims.

## Configuration

### Required

```env
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/<id>/<token>
MAX_PRICE=50
```

Create the webhook in Discord under **Channel Settings → Integrations → Webhooks**. Treat it as a secret and never commit `.env`.

### Legacy single-search mode

These settings are compatibility options. They are used only when no enabled searches have been added through the API.

```env
SEARCH_QUERY=
VINTED_BASE_URL=https://www.vinted.de
CATALOG_IDS=1242
SIZE_IDS=782,783,784,785,786,787,788,789,790
BRAND_IDS=53,1195
CURRENCY=EUR
MIN_PRICE=0
MAX_PRICE=50
RATE_LIMIT_MS=15000
```

### Evaluation and polling

```env
# shadow is the safe default: evaluations are logged, but legacy alerts continue.
# qualified suppresses listings that fail the model/size/condition/profit gate.
OPPORTUNITY_MODE=shadow
RESALE_RULES=Nike P-6000=35,Nike Air Max 95=40,Nike Air Max 90=35,Nike Air Max 97=40,Nike Shox=35
MIN_PROFIT_EUR=13
SEARCH_CONCURRENCY=2
```

Before using `OPPORTUNITY_MODE=qualified`, configure resale rules or Google Sheets history and confirm the condition-analysis policy. Missing or unknown condition is intentionally not qualified.

### Google Sheets history (optional)

Set all three values to enable synchronization; leaving them blank disables it and uses `RESALE_RULES` fallbacks.

```env
GOOGLE_SERVICE_ACCOUNT_JSON={"type":"service_account",...}
GOOGLE_SHEET_ID=your-spreadsheet-id
GOOGLE_WORKSHEET=Sales
HISTORY_SYNC_INTERVAL_MIN=60
```

Share the spreadsheet with the service account email as **Viewer**. The first row must contain these columns:

```text
model,size,condition,purchase_price,sale_price,costs,purchased_at,sold_at,source
```

`model`, `purchase_price`, and `sale_price` are required. Dates accept `YYYY-MM-DD`, RFC3339, or `DD.MM.YYYY`. Invalid rows are reported and do not replace the last valid snapshot.

### API protection

Set `API_TOKEN` to protect the search-management API. An empty token is intended only for local development.

```env
API_TOKEN=replace-with-a-long-random-token
```

## Managing searches

The API is available at `http://localhost:8080`. Search URLs must use HTTPS and a supported Vinted host.

For local development, open [http://localhost:8080/admin](http://localhost:8080/admin) to manage searches and inspect recent listings. The page also shows search health, last successful checks, and recent errors. When `API_TOKEN` is set, send the bearer token with API requests; an empty token is convenient for a local-only admin page.

```bash
export API_TOKEN=replace-with-a-long-random-token

curl -H "Authorization: Bearer $API_TOKEN" \
  http://localhost:8080/searches

curl -X POST http://localhost:8080/searches \
  -H "Authorization: Bearer $API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Nike P-6000 42","url":"https://www.vinted.de/catalog?search_text=Nike%20P-6000&size_ids=782","enabled":true,"priority":10}'

curl -X PUT http://localhost:8080/searches/<id> \
  -H "Authorization: Bearer $API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Nike P-6000 42","url":"https://www.vinted.de/catalog?search_text=Nike%20P-6000&size_ids=782","enabled":true,"priority":10}'

curl -X PATCH http://localhost:8080/searches/<id>/enabled \
  -H "Authorization: Bearer $API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"enabled":false}'

curl -X DELETE -H "Authorization: Bearer $API_TOKEN" \
  http://localhost:8080/searches/<id>
```

When at least one enabled API search exists, the worker polls those searches independently. Otherwise it uses the legacy environment settings.

Each polling cycle combines duplicate results by marketplace and listing ID, retains every search that found the listing, persists the record, then claims and sends one Discord notification. A failed Discord send releases the Redis claim so the listing can be retried.

## Architecture

- **Admin/API:** serves the authenticated search-management API and the operational admin page.
- **Worker:** polls enabled searches independently, normalizes and deduplicates listings, persists attribution, evaluates optional enrichment, and sends Discord alerts.
- **Redis:** suppresses duplicate deliveries.
- **PostgreSQL:** stores search configuration, normalized listings, search attribution, notification delivery records, and validated sales-history snapshots.
- **Google Sheets sync:** reads history periodically; it is never queried per listing.
- **Optional evaluation:** profitability and historical analysis never gate discovery in the default `shadow` mode.

## Verification and troubleshooting

```bash
go test ./...
docker compose build worker api
docker compose ps
docker compose logs --tail=100 worker api
curl --max-time 3 http://localhost:8080/admin
curl --max-time 3 http://localhost:8080/searches
```

The project expects tests and builds to run inside Docker when Go is not installed on the host. Use `docker run --rm -v "$PWD:/app" -w /app golang:1.24-alpine go test ./...` in that case. PostgreSQL migrations are applied on fresh Compose volumes, and additive MVP columns are checked at service startup for existing deployments. If the worker exits immediately, check `DISCORD_WEBHOOK_URL`, `MAX_PRICE`, PostgreSQL availability, and the worker logs. If the API is unreachable, check `docker compose ps api` and its logs.

## Data retention and safety

Normalized listing metadata, search attribution, and notification audit records are durable in PostgreSQL. Raw marketplace payloads are not retained by the MVP. Redis deduplication expires after 30 days. The bot never purchases items, messages sellers, negotiates, pays, or lists products automatically.
