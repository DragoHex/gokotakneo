# Phase 1: Foundation, Transport, Models & Error Taxonomy

Part of Gokiteconnect-parity Go SDK for Kotak-Neo (`gokotakneo`).

---

## 1. Scope
- Base `Client` struct with dynamic gateway discovery (`BaseURL`, `FeedURL`, `RTURL`).
- HTTP Transport seam (`HTTPClient` interface) with keep-alive connection pooling (`MaxIdleConnsPerHost: 50`, `ResponseHeaderTimeout: 5s`).
- Form-encoded `jData` serialization layer (stringified URL-encoded JSON payloads).
- Header injector: `Auth`, `Sid`, `neo-fin-key`, `Authorization`, `accept: application/json`.
- Modern Go 1.13+ Error taxonomy (`APIError`, `ErrAuthExpired`, `ErrInsufficientFunds`, `ErrOrderNotModifiable`, `ErrBrokerUnavailable`).
- Strongly-typed models with `shopspring/decimal.Decimal` and custom timestamp unmarshaler (`BrokerTime`).

---

## 2. Test-Driven Development (TDD) Test Matrix

Tests MUST be written and fail first (`errors_test.go`, `transport_test.go`, `time_test.go`) before implementation:

### A. Error Taxonomy (`errors_test.go`)
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive** | Kotak returns error JSON with `stat: "Not_Ok"`, `stCode: 1004`, `emsg: "Insufficient balance"` | Returns `*APIError` with Code `1004`, `errors.Is(err, ErrInsufficientFunds) == true`. |
| **Positive** | Kotak returns `stat: "Not_Ok"`, `stCode: 1006`, `emsg: "Order cannot be modified as it is already executed"` | Returns `*APIError` with Code `1006`, `errors.Is(err, ErrOrderNotModifiable) == true`. |
| **Positive** | Gateway returns HTTP 401 Unauthorized with `{status: "error", errorCode: "401"}` | Returns `*APIError` with `errors.Is(err, ErrAuthExpired) == true`. |
| **Negative** | Gateway returns HTTP 502 Bad Gateway with raw HTML string body | Returns synthetic `*APIError` wrapping `ErrBrokerUnavailable`, containing raw body snippet. |
| **Negative** | Kotak returns HTTP 200 OK but payload contains `stat: "Not_Ok"` | Correctly identifies failure despite HTTP 200 status code. |
| **Edge Case** | Malformed JSON or truncated error body | Returns fallback error without panic or nil-pointer dereference. |
| **Edge Case** | Error response with empty or missing `emsg` | Provides default descriptive message based on `stCode`. |

### B. Transport & `jData` Serialization (`transport_test.go`)
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive** | `ExecuteJData` called with Go struct payload | Request body is `jData=<url_encoded_json>`, `Content-Type: application/x-www-form-urlencoded`. |
| **Positive** | Standard request headers injection | Injects `Auth`, `Sid`, `neo-fin-key: neotradeapi`, `accept: application/json`. |
| **Negative** | Unmarshalable struct passed to `ExecuteJData` (e.g. channel field) | Returns error immediately without network call. |
| **Negative** | HTTP client returns network timeout | Wraps and returns `ErrBrokerUnavailable`. |
| **Edge Case** | Special characters in strings (e.g. quotes, ampersands, unicode in order tag) | Properly URL-encodes nested JSON without corrupting payload. |
| **Edge Case** | Dynamic base URL change | Requests route to new `BaseURL` immediately without connection leakage. |

### C. Timestamp Deserialization (`time_test.go`)
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive** | Kotak standard datetime `"2022/06/21 15:11:02"` | Parsed into exact `time.Time` struct. |
| **Positive** | ISO8601 `"2022-06-21T15:11:02+05:30"` | Parsed into exact `time.Time` struct. |
| **Positive** | Date only `"2022-06-21"` | Parsed into midnight `time.Time`. |
| **Edge Case** | `null` or empty string `""` | Sets zero time without unmarshaling error. |
| **Negative** | Completely invalid date string `"invalid-date"` | Returns descriptive parsing error. |
