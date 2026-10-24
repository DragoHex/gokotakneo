# Master TDD Test Matrix & Protocol Specification

Definitive verification guide for `gokotakneo`. Enforces pure Test-Driven Development (TDD) where test cases represent strict protocol specifications, preventing accidental test-fitting to faulty implementations.

---

## 1. Test-Driven Development Rules
1. **Red -> Green -> Refactor**:
   - Write tests first against the protocol specification (Kotak-Neo documentation).
   - Run tests to confirm they fail with compilation or assertion errors.
   - Implement the minimum code necessary to satisfy tests.
   - Refactor while keeping tests green.
2. **Never Fit Tests to Buggy Implementations**:
   - Tests assert exact HTTP methods, headers (`Sid`, `Auth`, `neo-fin-key`), status codes, and little-endian binary formats.
   - If an implementation produces different behavior than the spec, the implementation must be fixed, never the test.
3. **No External Network Dependencies in Unit Tests**:
   - All tests use hand-crafted HTTP mock responders (`httptest.Server` or fake `HTTPClient`) and mock WebSocket servers.

---

## 2. Comprehensive Test Suite Overview

| Test File | Target Package | Key Verifications |
| :--- | :--- | :--- |
| `errors_test.go` | `gokotakneo` | Error code taxonomy (1004, 1006, 401, 422, 500), `errors.Is` unwrapping, HTML 502 fallback. |
| `transport_test.go` | `gokotakneo` | Header injection, `jData` URL-encoded JSON marshaling, keep-alives, base URL switching. |
| `time_test.go` | `gokotakneo/models` | Non-standard datetime parsing (`2022/06/21 15:11:02`), ISO8601, null/empty handling. |
| `auth_test.go` | `gokotakneo` | 2-step TOTP + MPIN handshake, dynamic `baseUrl`/`feedUrl`/`rtUrl` re-binding, invalid credential rejections. |
| `orders_test.go` | `gokotakneo` | Place, modify, cancel orders across all varieties, price decimal formatting, RMS margin failure (1004). |
| `reports_test.go` | `gokotakneo` | Order book, order history timelines, trade book parsing, empty response handling. |
| `portfolio_test.go` | `gokotakneo` | Net/day positions, Demat holdings, account cash & margin limits parsing. |
| `market_test.go` | `gokotakneo` | Multi-symbol quotes, filter suffixes (`/all`, `/ltp`, `/ohlc`), historical data. |
| `feed_test.go` | `gokotakneo/feed` | WebSocket `native_batch` binary de-framing, little-endian unpacking, exchange divider scaling. |
| `orderstream_test.go` | `gokotakneo/orderstream` | Raw string handshake, live order execution postback handling, watchdog reconnect loop. |

---

## 3. Automated Verification Commands
```sh
# Run all unit tests
cd gokotakneo && go test -v -race ./...

# Run specific phase test
cd gokotakneo && go test -v ./... -run TestErrorTaxonomy
cd gokotakneo && go test -v ./... -run TestNativeBatchBinaryParser

# Verify code formatting & linting
cd gokotakneo && go vet ./...

# Verify 1:1 endpoint parity
python3 skills/go-sdk-parity-builder/scripts/audit_parity.py --go-dir gokotakneo
```
