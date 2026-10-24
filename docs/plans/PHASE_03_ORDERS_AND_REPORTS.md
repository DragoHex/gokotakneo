# Phase 3: Order Execution & Reports

Part of Gokiteconnect-parity Go SDK for Kotak-Neo (`gokotakneo`).

---

## 1. Scope
- **Place Order**: `POST <Base URL>/quick/order/rule/ms/place` with URL-encoded `jData`.
  - Supports: Regular (`am="NO"`), AMO (`am="YES"`), Products (`CNC`, `MIS`, `NRML`, `MTF`), Types (`MKT`, `L`, `SL`, `SL-M`), Validities (`DAY`, `IOC`).
- **Modify Order**: `POST <Base URL>/quick/order/vr/modify` with URL-encoded `jData`.
  - Modifies: `qt` (qty), `pr` (price), `tp` (trigger price), `pt` (order type), `vd` (validity).
- **Cancel Order**: `POST <Base URL>/quick/order/cancel` with URL-encoded `jData`.
  - Cancels open limit/SL orders by order number (`on`).
- **Order Reports**:
  - Order Book: `GET <Base URL>/quick/user/orders`
  - Order by ID: `GET <Base URL>/quick/user/orders/{order_no}`
  - Order History: `POST <Base URL>/quick/user/order/history` (`jData={"nOrdNo":"..."}`)
  - Trade Book: `GET <Base URL>/quick/user/trades`

---

## 2. Test-Driven Development (TDD) Test Matrix (`orders_test.go` & `reports_test.go`)

### A. Place Order
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive: Market Order** | Place market buy order (`pt="MKT"`, `pr=0`, `tt="B"`) | Encodes `jData` with `pr="0"`, returns `OrderResponse{OrderNumber: "...", Status: "Ok"}`. |
| **Positive: Limit Order** | Place limit sell order (`pt="L"`, `pr=150.25`, `tt="S"`) | Price formatted exactly to 2 decimal places (`pr="150.25"`), returns order number. |
| **Positive: SL-Limit Order** | Place stoploss limit order with trigger price (`tp=148.00`, `pr=147.50`) | Encodes both `tp` and `pr` in `jData`. |
| **Negative: Insufficient Margin** | Kotak returns `{stat: "Not_Ok", stCode: 1004, emsg: "Insufficient balance."}` | Returns `*APIError` with code `1004`, `errors.Is(err, ErrInsufficientFunds) == true`. |
| **Negative: Invalid Exchange Segment** | Segment specified as `"nse"` instead of `"nse_cm"` or `"nse_fo"` | Client input validator detects invalid segment and fails fast before network request. |
| **Edge Case: Zero Quantity** | Order placed with `Quantity <= 0` | Fails fast with `ErrInvalidInput`. |
| **Edge Case: Decimal Precision** | Fractional price with extra decimal digits (e.g. `123.4567`) | Formats according to tick size / broker decimal specification. |

### B. Modify & Cancel Order
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive: Modify Qty & Price** | Modify open order quantity and price | Sends `no`, `qt`, `pr` in `jData`, returns success status code 200. |
| **Positive: Cancel Order** | Cancel pending order | Sends `on: "<order_id>"`, `am: "NO"`, returns cancelled order number. |
| **Negative: Already Executed** | Kotak returns `{stat: "Not_Ok", stCode: 1006, emsg: "Order cannot be modified as it is already executed."}` | Returns `*APIError` with code `1006`, `errors.Is(err, ErrOrderNotModifiable) == true`. |
| **Negative: Already Cancelled** | Cancel called on already cancelled order | Returns `*APIError` with error diagnostic message. |
| **Edge Case: Empty Order ID** | Modify or Cancel called with empty string `""` | Fails fast with `ErrInvalidInput`. |

### C. Order Reports & Trade Book
| Case Type | Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **Positive: Order Book Listing** | `GetOrders` returns list of active and completed orders | Unmarshals array into `[]Order` with parsed timestamps and decimal average prices. |
| **Positive: Order History** | `GetOrderHistory` returns state transition records | Returns full timeline (e.g. `TRIGGER PENDING` -> `OPEN` -> `COMPLETE`). |
| **Positive: Trade Book** | `GetTrades` returns executions | Parses trade executions into `[]Trade` with executed quantity and fill prices. |
| **Negative: Order Not Found** | Order ID does not exist | Returns `ErrNotFound`. |
| **Edge Case: Empty Order Book** | Account has placed 0 orders today | Returns empty slice `[]Order{}` rather than nil or error. |
