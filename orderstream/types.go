package orderstream

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderUpdate represents an asynchronous order state transition event.
type OrderUpdate struct {
	OrderNumber     string          `json:"nOrdNo"`
	TradingSymbol   string          `json:"trdSym"`
	Status          string          `json:"ordSt"`
	FilledQuantity  int             `json:"fldQty"`
	AveragePrice    decimal.Decimal `json:"avgPrc"`
	RejectionReason string          `json:"rejRsn,omitempty"`
}

// PositionUpdate represents a live position exposure update.
type PositionUpdate struct {
	TradingSymbol string          `json:"trdSym"`
	Product       string          `json:"prod"`
	NetQuantity   int             `json:"netQty"`
	MTM           decimal.Decimal `json:"mtm"`
}

// StreamCallbacks represents callbacks for live order/position events.
type StreamCallbacks struct {
	OnConnect        func()
	OnOrderUpdate    func(OrderUpdate)
	OnPositionUpdate func(PositionUpdate)
	OnError          func(error)
	OnClose          func(code int, reason string)
	OnReconnect      func(attempt int, delay time.Duration)
}
