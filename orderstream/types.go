package orderstream

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderUpdate represents an asynchronous order state transition event.
type OrderUpdate struct {
	OrderNumber      string          `json:"nOrdNo"`
	ExchangeOrderID  string          `json:"exOrdId"`
	TradingSymbol    string          `json:"trdSym"`
	Symbol           string          `json:"sym"`
	InstrumentToken  string          `json:"tok"`
	ExchangeSegment  string          `json:"exSeg"`
	TransactionType  string          `json:"trnsTp"`
	ProductCode      string          `json:"prod"`
	PriceType        string          `json:"prcTp"`
	Status           string          `json:"ordSt"`
	Quantity         int             `json:"qty"`
	FilledQuantity   int             `json:"fldQty"`
	UnfilledQuantity int             `json:"unFldSz"`
	AveragePrice     decimal.Decimal `json:"avgPrc"`
	OrderTimestamp   string          `json:"ordDtTm"`
	RejectionReason  string          `json:"rejRsn,omitempty"`
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
