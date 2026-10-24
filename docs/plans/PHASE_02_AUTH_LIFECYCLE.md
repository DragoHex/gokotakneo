# Phase 2: Authentication & Session Lifecycle

Part of Gokiteconnect-parity Go SDK for Kotak-Neo (`gokotakneo`).

---

## 1. Scope
- Step 1: `tradeApiLogin` (`POST https://mis.kotaksecurities.com/login/1.0/tradeApiLogin`).
  - Inputs: `Authorization: <portal_token>`, `neo-fin-key: neotradeapi`, `mobileNumber`, `ucc`, `totp`.
  - Output: `ViewSession` (`token`, `sid`, `rid`, `kType: "View"`).
- Step 2: `tradeApiValidate` (`POST https://mis.kotaksecurities.com/login/1.0/tradeApiValidate`).
  - Inputs: `Authorization: <portal_token>`, `neo-fin-key: neotradeapi`, `sid: <viewSid>`, `Auth: <viewToken>`, `mpin`.
  - Output: `TradeSession` (`token`, `sid`, `baseUrl`, `feedUrl`, `rtUrl`, `greetingName`, `kType: "Trade"`).
- Dynamic Client Configuration:
  - Updates client `BaseURL`, `FeedURL`, `RTURL`.
  - Automatically sets `Auth` (session token) and `Sid` (session sid) headers for subsequent trading API calls.
- Invalidate / Logout API.

---

## 2. Test-Driven Development (TDD) Test Matrix (`auth_test.go`)

Tests must be written and executed against mock HTTP transport before code implementation:

| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive: Full Flow** | Step 1 called with valid TOTP, followed by Step 2 with valid MPIN | Returns `TradeSession`, client base URL updated to returned `baseUrl`, `client.IsAuthenticated() == true`. |
| **Positive: Header Injection** | Subsequent call after Step 2 | Outgoing HTTP requests carry `Auth: <tradeToken>` and `Sid: <tradeSid>` and `neo-fin-key: neotradeapi`. |
| **Negative: Invalid TOTP** | Step 1 receives HTTP 401 `{status: "error", message: "Invalid credentials or TOTP.", errorCode: "401"}` | Fails fast, returns `APIError` wrapping `ErrAuthExpired`. Client unauthenticated. |
| **Negative: Invalid MPIN** | Step 2 receives HTTP 401 `{status: "error", message: "Invalid MPIN.", errorCode: "401"}` | Fails fast, returns `APIError` wrapping `ErrAuthExpired`. `client.IsAuthenticated() == false`. |
| **Negative: Expired View Token** | Step 2 called with stale/expired `viewToken` | Returns `ErrAuthExpired`. |
| **Edge Case: Leading/Trailing Whitespace** | `mobileNumber`, `ucc`, `totp`, or `mpin` contains accidental whitespace | Automatically sanitizes (trims) input strings before network dispatch. |
| **Edge Case: BaseURL Scheme** | `baseUrl` or `feedUrl` returned with missing trailing slash or `http://` instead of `https://` | Normalizes URL cleanly without malforming subsequent API path concatenations. |
| **Edge Case: Missing Gateways** | Response missing `feedUrl` or `rtUrl` | Safely parses profile data without nil-pointer dereferences; streaming clients report clear error if attempted. |
