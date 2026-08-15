# Product Requirements Document

## Vinted Listing Monitor & Reselling Assistant

**Version:** 2.0  
**Status:** MVP  
**Primary Goal:** Monitor configurable Vinted search URLs, surface every matching new listing quickly, and allow search links to be managed through a simple admin web page.

---

## 1. Product Overview

The product is a Vinted listing-monitoring application for sneaker reselling.

The MVP should focus first on **reliable listing discovery and fast notifications**, not on automatically deciding whether a listing is profitable.

The core principle is:

> **If a listing appears in one of the configured Vinted search URLs, the application should detect it and surface it.**

The Vinted search URL itself defines the sourcing criteria.

Examples of criteria already encoded in a Vinted search URL may include:

- Brand
- Model / keyword
- Size
- Price range
- Condition
- Country
- Category
- Sorting
- Other marketplace filters supported by Vinted

The application should not add mandatory profitability or AI filtering on top of those searches.

---

## 2. Core MVP Workflow

```text
Admin adds Vinted search URL
        ↓
Application monitors enabled search
        ↓
New listing appears
        ↓
Listing is collected
        ↓
Duplicate check
        ↓
Discord notification
        ↓
User opens listing and decides what to do
```

The default MVP behavior is:

> **Configured search match → notify**

No profitability threshold is required for a listing to be shown.

---

## 3. Multi-Search Source Management

Supporting multiple Vinted search URLs is a **core MVP requirement**.

The user must be able to monitor multiple searches simultaneously.

Each search represents an independent Vinted sourcing strategy.

Examples:

```text
P-6000 Germany
[Vinted search URL]

Air Max 95 Germany
[Vinted search URL]

Nike Shox Germany
[Vinted search URL]

Broad Nike Search
[Vinted search URL]
```

All enabled searches should run independently and feed results into the same listing pipeline.

There should be no product requirement that forces all searches to use the same model, price range, size range, or strategy.

---

## 4. Admin Web Page

A simple **admin web page is part of the MVP**.

The main purpose of the admin page is to make search management possible without changing code or configuration files manually.

### 4.1 Search Management

The user should be able to:

- Add a Vinted search URL
- Give the search a name
- Enable or disable a search
- Edit an existing search
- Delete a search
- Optionally add notes
- Optionally set a priority
- See when the search was last checked
- See whether the search is healthy or experiencing an error

The intended experience should be as simple as:

```text
Sourcing Searches

✅ P6000 Germany
https://www.vinted.de/...

✅ Air Max 95 Germany
https://www.vinted.de/...

❌ Experimental Shox Search
https://www.vinted.de/...

[ + Add Search ]
```

Adding a new sourcing strategy should conceptually require only:

> **Paste Vinted search URL → Give it a name → Save**

### 4.2 Admin Page Scope for MVP

The admin page does not need to be a complex analytics dashboard.

For the MVP it only needs to provide practical operational controls.

Minimum areas:

```text
Searches
Listings
Historical Data
System Status
Optional Settings
```

The design should prioritize simplicity over visual complexity.

---

## 5. Listing Discovery

The application should continuously check all enabled Vinted search URLs for new listings.

The application should aim to detect new listings as quickly as reasonably possible.

Every listing returned by an enabled search should be eligible to enter the notification pipeline.

The system should preserve information about which search discovered the listing.

Example:

```text
Found through: P6000 Germany
```

If multiple searches find the same listing, all relevant search sources may be retained.

---

## 6. Deduplication

The same marketplace listing may appear in multiple configured searches.

The application must recognize duplicate listings and avoid sending duplicate notifications unnecessarily.

Conceptually:

```text
Search A ─┐
Search B ─┤
Search C ─┘
     ↓
Collect listings
     ↓
Deduplicate by listing identity
     ↓
Notify once
```

A listing may still retain attribution to every search that discovered it.

---

## 7. Listing Data

Where available, the application should collect and display:

- Listing ID
- Listing URL
- Title
- Description
- Price
- Currency
- Brand
- Size
- Condition
- Seller username
- Seller avatar
- Seller rating
- Seller review count
- Country / marketplace
- Original listing images
- Listing creation or update time
- Search source
- First detected time

The application should preserve raw marketplace information where practical so that future analysis features can use it later.

---

## 8. Discord Notifications

Discord notifications are a **core MVP feature**.

The notification should closely follow the visual structure of the provided Locker-style reference.

The priority is to make listings fast to scan and visually familiar.

### 8.1 Notification Structure

Conceptually:

```text
Seller avatar + seller username

🇮🇹 Clickable Listing Title | €8.90

Listing description excerpt...
[see more on Vinted...]

📅 Updated          📏 Size           🏷️ Brand
9 minutes ago       43.5              adidas

📦 Condition        🌟 Rating         💰 Price
Very good           ⭐⭐⭐⭐⭐ (2)       €8.90

[ ORIGINAL LISTING PHOTO COLLAGE / GRID ]

🚚 Direct listing link
```

### 8.2 Required Notification Content

Where available, include:

- Seller username
- Seller avatar
- Country flag
- Listing title
- Listing price
- Clickable listing link
- Description excerpt
- Updated / posted time
- Size
- Brand
- Condition
- Seller rating
- Review count
- Original Vinted photographs
- Direct link to the listing
- Search name that found the listing

### 8.3 Original Listing Images

The original Vinted photographs should be displayed prominently.

The application should preserve the reference layout as closely as Discord allows, including a collage/grid-like presentation when possible.

The application must never replace the seller's original photographs with AI-generated images.

---

## 9. Notification Philosophy

For the MVP, the application should **not require a profitability threshold before sending a notification**.

If a listing matches an enabled Vinted search URL and is new, it should normally be surfaced.

This is intentional because:

- The Vinted URL already contains the sourcing filters.
- The business may not yet have enough historical data for reliable profitability models.
- Missing good listings is worse than receiving some listings that later turn out to be uninteresting.
- The user should remain in control of the buying decision.

Optional filters may be introduced later, but they must not be mandatory to use the application.

---

## 10. Optional Profitability Analysis

Profitability analysis should exist as an **optional feature**, not as a core requirement for listing discovery.

The application must function fully even when profitability analysis is disabled.

When enabled in the future, profitability analysis may provide:

- Expected resale price
- Expected profit
- ROI
- Maximum purchase price
- Expected days to sell
- Deal Score
- Confidence

However:

> **Listings should not depend on this analysis in order to be discovered or displayed.**

The user should be able to choose between modes such as:

```text
Notification Mode:
○ Show all listings from configured searches
○ Show all listings + optional analysis
○ Future: Filter using analysis rules
```

The first option should be the default MVP behavior.

---

## 11. Optional Historical Data

Historical sales data should be treated as an optional enhancement.

The MVP must not assume that enough historical data already exists.

If historical data is unavailable:

```text
Listing discovery → still works
Discord notifications → still work
Admin search management → still works
```

Historical data may later improve:

- Resale estimates
- Model-specific pricing
- Size desirability
- Sell-through estimates
- Profit estimates
- Deal scoring
- Model discovery

The absence of historical data must never prevent the application from operating.

---

## 12. Optional Google Sheets Comparable-Sales Reference

Google Sheets should act as an **optional, non-blocking historical reference layer**.

The application must not require Google Sheets in order to monitor Vinted listings or send Discord notifications.

The user's existing Google Sheet should remain the source of truth for historical sales data.

The application should adapt to the user's current tracking format rather than forcing the user to redesign the sheet before the MVP can work.

### 12.1 Current Historical Data Format

The application should support the historical fields already being tracked by the user, including where available:

- Shoe / model name
- Size
- Purchase price
- Expected sale price
- Actual sale price
- Profit
- Margin / ROI
- Purchase date
- Sale date
- Days to sell
- Condition
- Any existing notes or related fields that can help identify the item

Not every row is required to contain every field.

The application should tolerate incomplete historical rows and use only the fields that are available and trustworthy.

### 12.2 Google Sheet Configuration in Admin Page

The Google Sheet connection must be configurable through the admin web page.

The Google Sheet URL must **not be hardcoded**.

The admin page should allow the user to:

- Paste or replace the Google Sheet URL
- Select or enter the worksheet / tab name
- Enable or disable historical-data sync
- Configure or review column mappings when necessary
- Trigger a manual sync
- See the last successful sync time
- See the number of rows successfully imported
- See sync errors or malformed rows
- Disconnect the current sheet
- Replace the sheet later without changing application code

Conceptually:

```text
Historical Data

Google Sheet URL
[ https://docs.google.com/spreadsheets/d/... ]

Worksheet / Tab
[ Sales ]

Sync
[ Enabled ]

Last synced
2 minutes ago

Imported rows
137

[ Sync Now ]    [ Save ]
```

Changing the Google Sheet link should be as simple as:

> **Open Admin → Replace Google Sheet URL → Save**

The next synchronization should use the newly configured sheet.

### 12.3 Synchronization

The application should periodically synchronize the configured Google Sheet into an internal normalized dataset or cache.

The application should **not query Google Sheets separately for every Vinted listing**.

Conceptually:

```text
Google Sheet
      ↓
Periodic sync
      ↓
Validation + normalization
      ↓
Internal historical cache
      ↓
Fast comparable-sales lookup
```

A manual refresh option should also be available from the admin page.

If the Google Sheet is temporarily unavailable, the core Vinted monitoring and Discord notification pipeline must continue operating.

### 12.4 Column Mapping

The application should support the user's existing column names and structure.

Where exact column names vary, the user should be able to map sheet columns to application fields through the admin page.

Example:

```text
Sheet column              Application field

Item / Shoe               Model
Size                      Size
Buying Price              Purchase price
Selling Price             Actual sale price
Profit                    Profit
ROI                       ROI
Date Bought               Purchase date
Date Sold                 Sale date
Days to Sell              Days to sell
Condition                 Condition
```

The system should store the mapping so it does not need to be reconfigured for every sync.

### 12.5 Historical Matching

When a new Vinted listing is discovered, the application may compare it against the synchronized historical dataset.

The matching process should prefer the most relevant comparables first.

Conceptually:

```text
1. Same normalized model + same size + similar condition
2. Same normalized model + nearby size + similar condition
3. Same normalized model + same size
4. Same normalized model across nearby sizes
5. Same normalized model generally
```

The matching strategy should be configurable and should not require an exact Vinted listing title match.

For example:

```text
Nike P6000 white silver
Nike P-6000 Damen
P 6000 Nike size 43
```

may all normalize to:

```text
Brand: Nike
Model: P-6000
Size: 43
```

### 12.6 Comparable-Sales Enrichment

If sufficiently relevant historical matches exist, the application may enrich the Discord notification with reference information.

Example:

```text
📊 Historical Reference

4 comparable sales found

Median sale price: €39.50
Observed sale range: €37–42
Median days to sell: 7.5 days
Average historical profit: €18.25

Current listing: €15
```

Historical information should be presented primarily as **evidence from previous sales**, especially while the dataset is still small.

The system should prefer language such as:

> "4 comparable historical sales had a median sale price of €39.50."

rather than:

> "This shoe will sell for €39.50."

### 12.7 Insufficient Historical Data

If no sufficiently relevant historical data exists, the listing should still be surfaced normally.

Example:

```text
📊 Historical Reference

No sufficiently similar previous sales found.
```

The absence of a historical match must never:

- Block the listing
- Delay the core notification unnecessarily
- Hide a listing that matched the configured Vinted search
- Be treated as evidence that the listing is bad

### 12.8 Relationship to Profitability Analysis

Historical data may later support optional calculations such as:

- Estimated resale value
- Expected profit
- ROI
- Expected time to sell
- Deal Score

However, these remain optional enrichment features.

The fundamental behavior remains:

```text
Vinted search match
      ↓
Listing is surfaced
      ↓
Historical data enriches it only when useful
```

### 12.9 Future Two-Way Sync

The initial integration should be one-way:

```text
Google Sheet → Application
```

A future version may optionally support:

```text
Application → Google Sheet
```

for writing back:

- Purchases
- Sale outcomes
- Actual sale prices
- Profit
- Days to sell

This future feature should not be required for MVP.

---

## 13. Optional Visual / Picture Analysis

AI picture evaluation should also be **optional**.

The application should work perfectly well without image analysis.

When enabled, picture analysis may inspect:

- Overall visual condition
- Creasing
- Sole wear
- Heel wear
- Stains
- Discoloration
- Visible damage
- Missing components
- Cleaning difficulty
- Photo completeness
- Model-identification clues
- Potential authenticity-risk signals
- Confidence

The application must not invent information that cannot be seen.

Example:

```text
Outsole condition: Unknown
Size label: Not visible
Visual confidence: 72%
```

### 13.1 Cost-Aware Analysis

If visual analysis is enabled, the application should avoid unnecessary AI costs.

Preferred flow:

```text
New listing
     ↓
Optional cheap/basic checks
     ↓
Optional low-cost vision
     ↓
Optional high-detail analysis only when needed
```

Higher-detail analysis should only be used when the user enables it or when a future analysis policy determines that the extra detail is worthwhile.

---

## 14. Optional Deal Score

A Deal Score from 0–100 may be introduced once enough data exists.

Possible inputs include:

- Expected profit
- ROI
- Demand
- Sell-through speed
- Visual condition
- Historical confidence
- Risk

This is **not required for the MVP**.

The application must not depend on the Deal Score for core monitoring and notifications.

---

## 15. Listing Status Tracking

The application should maintain a basic status for discovered listings.

Possible statuses:

- New
- Notified
- Viewed
- Interested
- Purchased
- Ignored
- Sold

For MVP, only the minimum states required to avoid duplicate notifications are necessary.

Additional workflow states may be added later.

---

## 16. System Status

The admin web page should provide basic operational visibility.

Useful information includes:

- Number of enabled searches
- Last successful check
- Searches currently failing
- Recently detected listings
- Recently sent notifications
- Basic errors
- Discord notification status

The goal is to make it easy to tell whether the monitor is functioning.

---

## 17. Speed Requirement

The product is intended for time-sensitive marketplace sourcing.

The application should therefore prioritize low listing-to-notification latency.

Useful timestamps to store include:

```text
Listing first detected
Listing processed
Duplicate check completed
Discord notification sent
```

The product should make it possible to measure and improve:

> **Listing detected → Discord alert sent**

Optional AI or historical analysis should not unnecessarily block a fast notification.

If an analysis step becomes slow, the system should be designed so the basic listing notification can still be delivered quickly.

---

## 18. Reliability

The application should handle temporary failures gracefully.

Examples:

- A configured search temporarily fails
- Discord is unavailable
- Marketplace response is incomplete
- Images cannot be loaded
- Optional AI service is unavailable
- Optional Google Sheets sync fails

Failures in optional systems must not bring down the core listing-monitoring pipeline.

---

## 19. MVP Scope

The MVP should include:

### Search Management

- Multiple Vinted search URLs
- Add / edit / delete searches
- Enable / disable searches
- Search naming
- Admin web page

### Listing Monitoring

- Monitor all enabled searches
- Detect new listings
- Preserve search attribution
- Deduplicate listings
- Store basic listing data

### Discord

- Locker-style Discord notification
- Original listing images
- Seller and listing metadata
- Direct Vinted link
- Fast delivery

### Admin

- Search-management interface
- Recent listings view
- Basic system status
- Error visibility

### Optional MVP Modules

These may be included but must remain optional:

- Google Sheets historical data sync
- Historical resale estimates
- Profitability calculations
- AI picture evaluation
- Deal Score

The application must remain fully usable when all optional analysis modules are disabled.

---

## 20. Out of Scope for Core MVP

The MVP does not need to:

- Automatically purchase listings
- Automatically negotiate with sellers
- Automatically message sellers
- Automatically make payments
- Automatically list inventory for resale
- Require AI to decide whether an item is good
- Require historical sales data
- Require Google Sheets
- Require a profitability model
- Require a Deal Score
- Fully automate the reselling business

---

## 21. MVP Success Criteria

The MVP is successful when it can consistently:

1. Allow the user to add a Vinted search URL through the admin web page.
2. Monitor multiple enabled Vinted searches.
3. Detect newly appearing listings.
4. Surface listings without requiring profitability analysis.
5. Avoid duplicate notifications for the same marketplace listing.
6. Send fast Discord notifications.
7. Display original listing photographs and important metadata.
8. Provide a direct link to the original listing.
9. Show which search discovered the listing.
10. Make it easy to enable, disable, edit, and remove searches.
11. Continue operating even when optional historical-data or AI features are disabled.

The primary MVP objective is:

> **Make Vinted sourcing faster by automatically watching the searches the user already knows how to configure.**

---

## 22. Long-Term Vision

The long-term product may evolve from a listing monitor into a personalized sourcing assistant.

Future intelligence may include:

- Historical resale estimates
- Profit predictions
- Sell-through predictions
- AI condition analysis
- Deal Score
- Search-performance analytics
- Inventory awareness
- Capital allocation
- Automatic purchase rules

However, the foundation should remain:

```text
User controls the searches
        ↓
Application reliably finds listings
        ↓
Application surfaces them quickly
        ↓
Optional intelligence helps the user decide
```

The intelligence layer should enhance the monitor rather than become a requirement for it to function.
