# Vinted Bot V2 🚀
> The fastest, most scalable Vinted monitoring bot. Built with **Go** & **Docker**.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/go-1.22+-00ADD8.svg?logo=go)

**Vinted Bot V2** is a complete rewrite of the popular Vinted monitor. It leverages the power of Go's concurrency and Redis's speed to check for new items faster than ever before.

## 🔥 Features

-   **⚡ Blazing Fast**: Written in Go for raw performance.
-   **🐳 Docker Ready**: One command to start everything (`docker-compose up`).
-   **🧠 Smart Alerts**: Only sends notifications for *new* items (Redis deduplication).
-   **🎨 Beautiful Embeds**: Rich Discord notifications with prices, photos, and direct links.
-   **🛡️ Anti-Ban**: Implements TLS fingerprinting and random jitter.

## 🚀 Getting Started

### Prerequisites
-   [Docker](https://www.docker.com/) & Docker Compose
-   A Discord Webhook URL

### Installation

1.  **Clone the repo**
    ```bash
    git clone https://github.com/yourusername/vinted-bot-v2.git
    cd vinted-bot-v2
    ```

2.  **Configure**
    Copy `.env.example` to `.env` and edit your settings:
    ```bash
    cp .env.example .env
    ```
    ```env
    DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
    SEARCH_QUERY="iPhone 13"
    VINTED_BASE_URL=https://www.vinted.de
    CATALOG_IDS=1242
    SIZE_IDS=782,783,784
    BRAND_IDS=53,1195
    MIN_PRICE=2
    MAX_PRICE=500
    RATE_LIMIT_MS=15000
    ```

3.  **Run**
    ```bash
    docker-compose up -d
    ```

That's it! The bot is now monitoring 24/7.

### Discord webhook

In Discord, open the destination channel's **Edit Channel** menu, then go to
**Integrations → Webhooks → New Webhook** and copy its URL into
`DISCORD_WEBHOOK_URL` in `.env`. Treat this URL like a password: do not commit
or share it. If it is ever exposed, delete that webhook in Discord and create a
new one.

The worker sends a Discord embed for each matching listing at or below
`MAX_PRICE`. Redis remembers delivered listings for 30 days, preventing repeat
alerts while the worker keeps polling.

To use a Vinted filtered search, copy its numeric category, size, and brand IDs
into comma-separated `CATALOG_IDS`, `SIZE_IDS`, and `BRAND_IDS` values. Leave
`SEARCH_QUERY` empty when the search has no text term.

## 🛠️ Architecture

-   **Worker**: Handles scraping and parsing.
-   **Redis**: Deduplicates items to prevent spam.
-   **API** (Coming Soon): REST endpoints to manage alerts.

## 🤝 Contributing

PRs are welcome! Let's make this the #1 Vinted bot on GitHub.
