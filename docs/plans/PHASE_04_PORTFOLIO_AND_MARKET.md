# Phase 4: Portfolio, Limits & Market Data

Part of Gokiteconnect-parity Go SDK for Kotak-Neo (`gokotakneo`).

---

## 1. Scope
- **Positions**: `GET <Base URL>/quick/user/positions`. Consolidated open & closed intraday/carry positions.
- **Holdings**: `GET <Base URL>/portfolio/v1/holdings`. Demat custody portfolio with P&L, MTM, holding cost.
- **Limits / Balances**: `POST <Base URL>/quick/user/limits` (`jData={"seg":"ALL","exch":"ALL","prod":"ALL"}`). Cash, collateral, margin used.
- **Quotes API**: `GET <Base URL>/script-details/1.0/quotes/neosymbol/<query>[/<filter>]`.
  - Queries: `<exchange_segment>|<instrument>` (comma-separated for multi-symbol).
  - Filters: `all`, `ltp`, `ohlc`, `depth`, `circuit_limits`, `52W`, `oi`, `scrip_details`.
- **Historical Data**: Historical candle bars.
- **Instruments Master**: Streaming CSV / JSON scrip master downloader.

---

## 2. Test-Driven Development (TDD) Test Matrix (`portfolio_test.go` & `market_test.go`)

### A. Positions & Holdings
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive: Open Positions** | Fetch position book with open CNC and MIS positions | Returns array of `Position` structs with parsed decimal buy/sell amounts, lot sizes, quantities. |
| **Positive: Demat Holdings** | Fetch long-term portfolio | Returns `[]Holding` with calculated `UnrealisedGainLoss`, holding cost, and market value. |
| **Edge Case: Empty Portfolio** | New account with 0 positions / holdings | Returns empty slice `[]Position{}` / `[]Holding{}` (no error). |
| **Negative: Unauthenticated Call** | Call without session token | Returns `ErrAuthExpired`. |

### B. Limits & Margins
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive: Full Account Limits** | Query limits with `seg="ALL"`, `exch="ALL"`, `prod="ALL"` | Returns `Limits` struct with decimal `Net`, `AvailableMargin`, `CollateralValue`, `MarginUsed`. |
| **Positive: Segment-Scoped Limits** | Query limits specifically for FO segment | Encodes `seg="FO"` in `jData`. |
| **Edge Case: Negative Balance** | Account has margin debit / negative cash | Correctly parses negative values into signed `decimal.Decimal` without underflow. |

### C. Market Quotes & Depth
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive: Full Quote Snapshot** | Query `nse_cm|INFY-EQ/all` | Returns `Quote` with full 5x5 buy/sell depth book, OHLC, volume, 52W high/low. |
| **Positive: Lightweight LTP** | Query `nse_cm|TCS-EQ/ltp` | Returns lightweight map with LTP only. |
| **Positive: Multi-Symbol Query** | Query `nse_cm|Nifty 50,bse_cm|SENSEX/ohlc` | Parses and returns keyed map for both instruments. |
| **Negative: Invalid Symbol** | Instrument does not exist in master | Returns `ErrNotFound`. |
| **Edge Case: Comma/Pipe Formatting** | Symbols containing special characters | Properly encodes URL path segments. |
