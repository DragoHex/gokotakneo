package feed

import (
	"time"

	"github.com/shopspring/decimal"
)

// Tick represents live market quote data parsed from the binary stream.
type Tick struct {
	InstrumentToken uint32          `json:"instrument_token"`
	ExchangeSegment string          `json:"exchange_segment"`
	TradingSymbol   string          `json:"trading_symbol"`
	LastPrice       decimal.Decimal `json:"last_price"`
	Volume          int64           `json:"volume"`
	Open            decimal.Decimal `json:"open"`
	High            decimal.Decimal `json:"high"`
	Low             decimal.Decimal `json:"low"`
	Close           decimal.Decimal `json:"close"`
	Depth           MarketDepth     `json:"depth"`
}

// MarketDepth represents 5-depth buy/sell order book.
type MarketDepth struct {
	Buy  [5]DepthItem `json:"buy"`
	Sell [5]DepthItem `json:"sell"`
}

// DepthItem represents single bid/ask price and quantity.
type DepthItem struct {
	Price    decimal.Decimal `json:"price"`
	Quantity int             `json:"quantity"`
	Orders   int             `json:"orders"`
}

// FeedCallbacks represents user-registered streaming event callbacks.
type FeedCallbacks struct {
	OnConnect   func()
	OnTick      func(Tick)
	OnError     func(error)
	OnClose     func(code int, reason string)
	OnReconnect func(attempt int, delay time.Duration)
}
