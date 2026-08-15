# Product Requirements Document

## Vinted Shoe Reselling Sourcing Assistant

**Version:** 1.0\
**Status:** MVP\
**Goal:** Scale shoe reselling to 50 completed sales per month

------------------------------------------------------------------------

## 1. Product Overview

Build an automated sourcing assistant that identifies potentially
profitable Nike shoe listings, evaluates their condition and resale
potential, and presents promising opportunities to the reseller in
Discord.

The system should reduce the time spent manually searching for inventory
while increasing the number of good purchasing opportunities found.

The system is an **assistant, not an autonomous purchasing system**. The
reseller makes the final purchasing decision.

------------------------------------------------------------------------

## 2. Problem

The main limitation of the current business is sourcing.

Good shoes at attractive prices do not appear consistently, and finding
them requires manually monitoring listings and evaluating:

-   Model
-   Size
-   Price
-   Condition
-   Expected resale value
-   Potential profit

The objective is to make this process significantly more scalable.

------------------------------------------------------------------------

## 3. Business Goal

The initial business goal is:

> **50 shoes successfully sold per month.**

The sourcing system should help generate enough suitable inventory to
support this target.

The system should prioritize **finding genuinely good opportunities
rather than limiting the number of notifications**.

Good opportunities can be relatively uncommon, so the system should err
toward surfacing potentially interesting listings and allow the reseller
to make the final decision.

------------------------------------------------------------------------

## 4. Target Products

The MVP focuses exclusively on **Nike shoes**.

Initial priority models:

-   Nike P-6000
-   Nike Air Max 95
-   Nike Air Max 90
-   Nike Air Max 97
-   Nike Shox

The system should eventually be capable of identifying additional
profitable Nike models based on historical business performance.

------------------------------------------------------------------------

## 5. Target Sizes

The MVP should focus on:

**EU 41--45**

The size range should be configurable so that it can be changed later.

------------------------------------------------------------------------

## 6. Profitability Requirements

The minimum desired net profit is:

> **€13 per shoe**

This represents profit after the user's current costs.

The system should calculate:

**Expected profit = Expected resale price − Purchase price − applicable
costs**

The minimum expected profit threshold is €13.

A listing that is expected to generate exactly €13 should still be
surfaced.

------------------------------------------------------------------------

## 7. Maximum Purchase Price

The system should calculate the maximum price the reseller should pay
while achieving the minimum target profit.

Example:

> Expected resale: €35\
> Minimum desired profit: €13
>
> **Maximum purchase price: €22**

If a listing costs €15:

> Expected profit: €20

The listing should therefore qualify.

------------------------------------------------------------------------

## 8. Model-Specific Pricing

The system should **not assume every Nike shoe has the same resale
value**.

Historical business data should be used to establish expected resale
prices for individual models.

For example:

  Model                   Potential resale
  --------------------- ------------------
  P-6000                         \~€30--40
  Air Max 90                     \~€30--40
  Air Max 95                    \~€30--40+
  Air Max 97                    \~€30--40+
  Shox                           \~€30--40
  Higher-value models       Model-specific

These values are examples rather than fixed rules.

The system should use historical evidence wherever sufficient data
exists.

------------------------------------------------------------------------

## 9. Historical Business Data

The user's existing sales history should be incorporated into the
product.

The historical data contains information such as:

-   Shoe model
-   Size
-   Purchase price
-   Expected sale price
-   Actual sale price
-   Profit
-   Margin
-   Days to sell

The data should be used to establish an initial understanding of the
business.

For example:

> "Air Max 95 historically sells for approximately €X."

> "Size 42 performs particularly well."

> "This model typically sells within X days."

Historical values should be treated as estimates rather than guarantees.

------------------------------------------------------------------------

## 10. Historical Price Estimation

When sufficient historical data exists for a model, the system should
use that data to estimate expected resale value.

The estimation should consider, where possible:

-   Model
-   Size
-   Condition
-   Historical sale prices
-   Number of historical sales

The system should prefer robust estimates over isolated outliers.

For example, a single €66 sale should not automatically mean that every
shoe of that model is valued at €66.

------------------------------------------------------------------------

## 11. Condition Assessment

The system should evaluate the condition of the shoe using the available
listing information and photographs.

The assessment should identify:

-   Overall condition
-   Visible wear
-   Sole wear
-   Creasing
-   Stains
-   Discoloration
-   Damage
-   Missing components
-   Cleaning difficulty

The system should provide a condition rating, for example:

-   Very Good
-   Good
-   Okay
-   Poor
-   Unknown

It should also provide a confidence level.

If the photographs are insufficient to determine condition, the system
should explicitly say so rather than making an overly confident
assessment.

------------------------------------------------------------------------

## 12. Condition and Profit Interaction

Condition should influence the expected resale price.

Example:

### Very good condition

Expected resale:

**€40**

### Good condition

Expected resale:

**€35**

### Okay condition

Expected resale:

**€30**

### Poor condition

Expected resale:

**€20**

These are initial assumptions and should eventually be replaced or
refined using historical data.

A particularly valuable model in excellent condition may justify a
significantly higher expected resale price.

------------------------------------------------------------------------

## 13. Opportunity Evaluation

Each listing should receive an evaluation containing at least:

-   Model
-   Size
-   Purchase price
-   Condition
-   Expected resale price
-   Expected profit
-   Maximum purchase price
-   ROI
-   Confidence

The core question should be:

> **"Is this listing likely to generate at least €13 of net profit?"**

------------------------------------------------------------------------

## 14. Listing Qualification

A listing should qualify as a potential opportunity when:

1.  It is a target Nike shoe.
2.  It is within the target size range.
3.  The listing contains sufficient information for evaluation.
4.  The expected resale value can reasonably be estimated.
5.  Expected net profit is at least €13.
6.  The condition is not considered unacceptable.

Listings that fail these criteria should normally not generate an alert.

------------------------------------------------------------------------

## 15. High-Value Exceptions

The system should not impose an unnecessarily low purchase-price
ceiling.

Some models may justify a higher purchase price because they have
substantially higher resale values.

For example:

> Model A\
> Expected resale: €35\
> Maximum purchase price: €22

versus:

> Model B\
> Expected resale: €60\
> Maximum purchase price: €47

Both can be valid opportunities.

The system should therefore optimize around **expected profit**, rather
than simply searching for cheap shoes.

------------------------------------------------------------------------

## 16. Duplicate Listings

The same listing should not repeatedly generate notifications.

The system should recognize previously processed listings and maintain
their status.

Possible statuses include:

-   New
-   Evaluated
-   Alerted
-   Interested
-   Purchased
-   Sold
-   Ignored

------------------------------------------------------------------------

## 17. Opportunity Notifications

Promising listings should be presented in Discord using a highly visual
format.

The notification should resemble the user's existing preferred
notification style.

### Example

**🇩🇪 Nike P-6000 \| €15**

> Nike P-6000 Metallic Silver

**Updated:** 2 minutes ago\
**Size:** 42\
**Brand:** Nike

**Condition:** Very Good\
**AI Condition:** 8.5/10\
**Seller:** ⭐⭐⭐⭐⭐ (71)

**Price:** €15\
**Expected resale:** \~€35

Then display the original listing photographs.

------------------------------------------------------------------------

## 18. Financial Analysis in Notification

The notification should clearly show:

> 💰 Expected resale: **€35**\
> 💵 Expected profit: **€20**\
> 🎯 Maximum purchase price: **€22**\
> 📈 ROI: **133%**

The user should be able to understand the opportunity without performing
additional calculations.

------------------------------------------------------------------------

## 19. AI Analysis in Notification

The notification should include a concise condition assessment.

Example:

> 🤖 **AI Condition Analysis --- 8.5/10**
>
> ✅ Upper appears clean\
> ✅ No major damage visible\
> ⚠️ Minor creasing\
> ⚠️ Light sole wear
>
> **Cleaning difficulty:** Low\
> **Confidence:** 91%

The detailed analysis should remain secondary to the actual photographs.

------------------------------------------------------------------------

## 20. Original Listing Images

The notification should display the original listing photographs
whenever available.

The reseller should be able to visually inspect the shoes directly from
Discord.

The system should **not rely solely on AI's interpretation of the
photographs**.

------------------------------------------------------------------------

## 21. Listing Link

Every notification must contain a direct link to the original listing.

The reseller should be able to open the listing immediately.

------------------------------------------------------------------------

## 22. User Actions

The notification should eventually support actions such as:

### View

Open the original listing.

### Interested

Mark the opportunity as interesting.

### Purchased

Mark the shoe as purchased.

### Ignore

Mark the opportunity as irrelevant.

These actions should update the listing's status for future analysis.

**The MVP must not automatically purchase shoes or perform account
actions on the marketplace.**

------------------------------------------------------------------------

## 23. Learning From Purchases

The system should eventually record what happened after an opportunity
was identified.

For purchased shoes:

-   Actual purchase price
-   Actual sale price
-   Actual profit
-   Days to sell
-   Condition
-   Model
-   Size

This should be compared against the original prediction.

Example:

> Predicted resale: €35\
> Actual resale: €37

> Predicted profit: €18\
> Actual profit: €20

This feedback should improve future estimates.

------------------------------------------------------------------------

## 24. Business Analytics

The system should eventually provide:

-   Listings evaluated
-   Opportunities detected
-   Opportunities purchased
-   Shoes sold
-   Total profit
-   Average profit
-   Median profit
-   ROI
-   Average days to sell
-   Profit by model
-   Profit by size
-   Profit by condition
-   Prediction accuracy

------------------------------------------------------------------------

## 25. Model Discovery

The initial system focuses on five Nike model groups.

Over time, the system should identify additional opportunities from
historical data.

For example:

> "Nike Zoom Fly 5 has generated unusually high profits."

The system could then recommend adding it to the sourcing targets.

The user should remain in control of which models become active sourcing
targets.

------------------------------------------------------------------------

## 26. Alert Philosophy

The system should prioritize **high-quality opportunities rather than
notification volume**.

However, because genuinely good deals are relatively uncommon, the
system should not impose an arbitrary daily alert limit.

The user is comfortable receiving many alerts.

The €13 expected-profit threshold is therefore the primary economic
filter.

------------------------------------------------------------------------

## 27. MVP Scope

The first version should support:

### Product discovery

-   Target Nike models
-   Sizes 41--45
-   New listing detection

### Evaluation

-   Model identification
-   Condition assessment
-   Historical resale estimation
-   Expected profit
-   Maximum purchase price
-   ROI

### Notification

-   Rich Discord notification
-   Listing photographs
-   Financial analysis
-   AI condition analysis
-   Direct listing link

### Data

-   Historical sales data
-   Previously seen listings
-   Evaluation results

### Manual decision

-   User decides whether to purchase

------------------------------------------------------------------------

## 28. Out of Scope for MVP

The first version should **not** attempt to:

-   Automatically purchase products
-   Automatically negotiate
-   Automatically message sellers
-   Automatically list products for resale
-   Automatically make payments
-   Circumvent marketplace protections
-   Build a complex dashboard
-   Support multiple marketplaces
-   Support multiple brands
-   Fully automate the business

------------------------------------------------------------------------

## 29. Success Criteria

The MVP should be considered successful when it can consistently:

1.  Find relevant Nike listings.
2.  Correctly identify target models.
3.  Evaluate shoe condition reasonably well.
4.  Estimate resale prices using historical business data.
5.  Identify listings with expected profit ≥ €13.
6.  Present those opportunities clearly in Discord.
7.  Avoid repeatedly alerting on the same listing.
8.  Allow the user to manually act on opportunities.
9.  Record outcomes for future improvement.

The ultimate business success metric is:

> **Enable the business to reach 50 completed shoe sales per month while
> reducing the amount of manual sourcing effort required per shoe.**

------------------------------------------------------------------------

## 30. Long-Term Vision

The long-term product should become a **personalized shoe-sourcing
assistant** that understands:

-   Which models are profitable
-   Which sizes perform best
-   Which conditions are worth buying
-   What each model can realistically sell for
-   How much the reseller should pay
-   Which opportunities are likely to sell quickly
-   Which purchases are risky

The system should increasingly learn from the reseller's actual results.

The long-term objective is not simply:

> **"Find cheap shoes."**

It is:

> **"Find the shoes that I am most likely to buy cheaply, resell
> successfully, and make good money on."**
