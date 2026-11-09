package orderstream

import (
	"strings"
	"time"

	"github.com/DragoHex/gokotakneo/models"
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

// SymbolName returns TradingSymbol if present, falling back to Symbol.
func (o *OrderUpdate) SymbolName() string {
	if strings.TrimSpace(o.TradingSymbol) != "" {
		return strings.TrimSpace(o.TradingSymbol)
	}
	return strings.TrimSpace(o.Symbol)
}

// NormalizedMCXQuantity returns the order quantity in standard lots for MCX contracts.
// For non-MCX exchanges, it returns Quantity unchanged.
func (o *OrderUpdate) NormalizedMCXQuantity() int {
	return models.NormalizeMCXQuantity(o.SymbolName(), o.ExchangeSegment, o.Quantity)
}

// NormalizedMCXFilledQuantity returns the filled quantity in standard lots for MCX contracts.
// For non-MCX exchanges, it returns FilledQuantity unchanged.
func (o *OrderUpdate) NormalizedMCXFilledQuantity() int {
	return models.NormalizeMCXQuantity(o.SymbolName(), o.ExchangeSegment, o.FilledQuantity)
}

// NormalizedStatus converts broker order statuses (complete, traded, rejected, cancelled)
// to uppercase canonical states (COMPLETE, REJECTED, CANCELLED).
func (o *OrderUpdate) NormalizedStatus() string {
	s := strings.ToUpper(strings.TrimSpace(o.Status))
	switch s {
	case "TRADED", "COMPLETE":
		return "COMPLETE"
	case "REJECTED", "REJECT":
		return "REJECTED"
	case "CANCELLED", "CANCEL":
		return "CANCELLED"
	default:
		return s
	}
}

// IsTerminal reports whether the order update represents a final terminal state.
func (o *OrderUpdate) IsTerminal() bool {
	st := o.NormalizedStatus()
	return st == "COMPLETE" || st == "REJECTED" || st == "CANCELLED"
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
	OnReconnected    func() // Invoked only upon successful reconnection after a prior disconnect
	OnOrderUpdate    func(OrderUpdate)
	OnPositionUpdate func(PositionUpdate)
	OnError          func(error)
	OnClose          func(code int, reason string)
	OnReconnect      func(attempt int, delay time.Duration) // Invoked before backoff delay during retry
}
