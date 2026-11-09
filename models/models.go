package models

import (
	"encoding/json"
	"strings"

	"github.com/shopspring/decimal"
)

// ViewSession represents Step 1 authentication output.
type ViewSession struct {
	Token        string `json:"token"`
	Sid          string `json:"sid"`
	Rid          string `json:"rid"`
	KType        string `json:"kType"`
	Status       string `json:"status"`
	GreetingName string `json:"greetingName"`
}

// TradeSession represents Step 2 authentication output with dynamic endpoints.
type TradeSession struct {
	Token        string `json:"token"`
	Sid          string `json:"sid"`
	Rid          string `json:"rid"`
	BaseURL      string `json:"baseUrl"`
	FeedURL      string `json:"feedUrl"`
	RTURL        string `json:"rtUrl"`
	KType        string `json:"kType"`
	Status       string `json:"status"`
	GreetingName string `json:"greetingName"`
	UCC          string `json:"ucc"`
}

// OrderParams represents input parameters for placing a Kotak-Neo order.
type OrderParams struct {
	Am string `json:"am"` // "NO" (regular), "YES" (AMO)
	Dq string `json:"dq"` // Disclosed quantity
	Es string `json:"es"` // Exchange segment ("nse_cm", "bse_cm", "nse_fo", etc.)
	Mp string `json:"mp"` // Market protection value
	Pc string `json:"pc"` // Product code ("CNC", "MIS", "NRML", "MTF")
	Pf string `json:"pf"` // Portfolio flag ("N")
	Pr string `json:"pr"` // Price for limit order ("0" for market)
	Pt string `json:"pt"` // Order type ("MKT", "L", "SL", "SL-M")
	Qt string `json:"qt"` // Quantity
	Rt string `json:"rt"` // Validity ("DAY", "IOC")
	Tp string `json:"tp"` // Trigger price
	Ts string `json:"ts"` // Trading symbol
	Tt string `json:"tt"` // Transaction type ("B" for buy, "S" for sell)
}

// OrderModifyParams represents input for modifying an open order.
type OrderModifyParams struct {
	OrderNo  string `json:"no"` // Original Nest Order Number
	Am       string `json:"am,omitempty"`
	Dq       string `json:"dq,omitempty"`
	Es       string `json:"es,omitempty"`
	Mp       string `json:"mp,omitempty"`
	Pc       string `json:"pc,omitempty"`
	Pr       string `json:"pr,omitempty"`
	Pt       string `json:"pt,omitempty"`
	Qt       string `json:"qt,omitempty"`
	Rt       string `json:"rt,omitempty"`
	Tp       string `json:"tp,omitempty"`
	Ts       string `json:"ts,omitempty"`
	Tt       string `json:"tt,omitempty"`
}

// OrderResponse represents order placement/modification/cancellation acknowledgement.
type OrderResponse struct {
	Stat        string `json:"stat"`
	StCode      int    `json:"stCode"`
	OrderNumber string `json:"nOrdNo"`
}

// Order represents an entry in the order book.
type Order struct {
	OrderNumber     string          `json:"nOrdNo"`
	ExchangeOrderID string          `json:"exOrdId"`
	TradingSymbol   string          `json:"trdSym"`
	Symbol          string          `json:"sym,omitempty"`
	ExchangeSegment string          `json:"exSeg"`
	TransactionType string          `json:"trnsTp"`
	ProductCode     string          `json:"prod"`
	PriceType       string          `json:"prcTp"`
	Status          string          `json:"ordSt"`
	Quantity        int             `json:"qty"`
	FilledQuantity  int             `json:"fldQty"`
	Price           decimal.Decimal `json:"prc"`
	AveragePrice    decimal.Decimal `json:"avgPrc"`
	TriggerPrice    decimal.Decimal `json:"trgPrc"`
	OrderTimestamp  BrokerTime      `json:"ordDtTm"`
	RejectionReason string          `json:"rejRsn,omitempty"`
}

// SymbolName returns TradingSymbol if present, falling back to Symbol.
func (o *Order) SymbolName() string {
	if strings.TrimSpace(o.TradingSymbol) != "" {
		return strings.TrimSpace(o.TradingSymbol)
	}
	return strings.TrimSpace(o.Symbol)
}

// NormalizedTransactionType normalizes Kotak transaction type (B/S) to canonical BUY/SELL.
func (o *Order) NormalizedTransactionType() string {
	switch strings.ToUpper(strings.TrimSpace(o.TransactionType)) {
	case "S", "SELL":
		return "SELL"
	default:
		return "BUY"
	}
}

// NormalizedOrderType normalizes Kotak price type (L/MKT/SL/SL-M) to canonical LIMIT/MARKET/SL/SL-M.
func (o *Order) NormalizedOrderType() string {
	switch strings.ToUpper(strings.TrimSpace(o.PriceType)) {
	case "L", "LIMIT":
		return "LIMIT"
	case "MKT", "MARKET":
		return "MARKET"
	case "SL":
		return "SL"
	case "SL-M", "SLM":
		return "SL-M"
	default:
		return "MARKET"
	}
}

// NormalizedProduct normalizes Kotak product code (CNC/NRML/MIS) to canonical CNC/NRML/MIS.
func (o *Order) NormalizedProduct() string {
	switch strings.ToUpper(strings.TrimSpace(o.ProductCode)) {
	case "CNC":
		return "CNC"
	case "NRML", "MARGIN":
		return "NRML"
	case "MIS", "INTRADAY":
		return "MIS"
	default:
		return strings.ToUpper(strings.TrimSpace(o.ProductCode))
	}
}

// NormalizedExchange normalizes Kotak exchange segment (nse_fo, bse_fo, mcx_fo, etc.) to canonical NFO/BFO/MCX/BSE/NSE.
func (o *Order) NormalizedExchange() string {
	s := strings.ToLower(strings.TrimSpace(o.ExchangeSegment))
	switch {
	case strings.HasPrefix(s, "nse_fo") || strings.HasPrefix(s, "nfo"):
		return "NFO"
	case strings.HasPrefix(s, "bse_fo") || strings.HasPrefix(s, "bfo"):
		return "BFO"
	case strings.HasPrefix(s, "mcx"):
		return "MCX"
	case strings.HasPrefix(s, "bse"):
		return "BSE"
	default:
		return "NSE"
	}
}

// IsTerminal returns true if the order is in a final terminal state.
func (o *Order) IsTerminal() bool {
	st := strings.ToUpper(strings.TrimSpace(o.Status))
	return st == "COMPLETE" || st == "REJECTED" || st == "CANCELLED"
}

// NormalizedMCXQuantity returns the order quantity converted from commodity units to standard lots for MCX contracts.
// For non-MCX exchanges, it returns Quantity untouched.
func (o *Order) NormalizedMCXQuantity() int {
	return NormalizeMCXQuantity(o.SymbolName(), o.NormalizedExchange(), o.Quantity)
}

// NormalizedMCXFilledQuantity returns the filled quantity converted from commodity units to standard lots for MCX contracts.
// For non-MCX exchanges, it returns FilledQuantity untouched.
func (o *Order) NormalizedMCXFilledQuantity() int {
	return NormalizeMCXQuantity(o.SymbolName(), o.NormalizedExchange(), o.FilledQuantity)
}

// Trade represents an executed fill from the trade book.
type Trade struct {
	TradeID         string          `json:"trdId"`
	OrderNumber     string          `json:"nOrdNo"`
	TradingSymbol   string          `json:"trdSym"`
	ExchangeSegment string          `json:"exSeg"`
	TransactionType string          `json:"trnsTp"`
	Quantity        int             `json:"qty"`
	Price           decimal.Decimal `json:"prc"`
	FillTimestamp   BrokerTime      `json:"trdDtTm"`
}

// Position represents an intraday or overnight position.
type Position struct {
	AccountID       string          `json:"actId"`
	Product         string          `json:"prod"`
	ExchangeSegment string          `json:"exSeg"`
	TradingSymbol   string          `json:"trdSym"`
	Symbol          string          `json:"sym"`
	Quantity        int             `json:"qty,string"`
	BuyAmount       decimal.Decimal `json:"buyAmt"`
	SellAmount      decimal.Decimal `json:"sellAmt"`
	CFBuyAmount     decimal.Decimal `json:"cfBuyAmt"`
	CFSellAmount    decimal.Decimal `json:"cfSellAmt"`
	CFBuyQty        int             `json:"cfBuyQty,string"`
	CFSellQty       int             `json:"cfSellQty,string"`
	FilledBuyQty    int             `json:"flBuyQty,string"`
	FilledSellQty   int             `json:"flSellQty,string"`
	BoardLotQty     int             `json:"brdLtQty"`
	SquareOffFlag   string          `json:"sqrFlg"`
	LotSize         int             `json:"lotSz,string"`
	StrikePrice     decimal.Decimal `json:"stkPrc"`
	UpdateTime      BrokerTime      `json:"hsUpTm"`
}

// UnmarshalJSON unmarshals Position JSON and automatically calculates Quantity if omitted.
func (p *Position) UnmarshalJSON(data []byte) error {
	type Alias Position
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*p = Position(aux)
	if p.Quantity == 0 {
		p.Quantity = p.NetQuantity()
	}
	return nil
}

// TotalBuyQty returns the aggregate buy quantity across carried forward and day executions.
func (p *Position) TotalBuyQty() int {
	return p.CFBuyQty + p.FilledBuyQty
}

// TotalSellQty returns the aggregate sell quantity across carried forward and day executions.
func (p *Position) TotalSellQty() int {
	return p.CFSellQty + p.FilledSellQty
}

// TotalBuyAmount returns the aggregate buy amount across carried forward and day executions.
func (p *Position) TotalBuyAmount() decimal.Decimal {
	return p.CFBuyAmount.Add(p.BuyAmount)
}

// TotalSellAmount returns the aggregate sell amount across carried forward and day executions.
func (p *Position) TotalSellAmount() decimal.Decimal {
	return p.CFSellAmount.Add(p.SellAmount)
}

// NetQuantity returns the open net quantity (TotalBuyQty - TotalSellQty).
func (p *Position) NetQuantity() int {
	net := p.TotalBuyQty() - p.TotalSellQty()
	if net == 0 && p.Quantity != 0 {
		return p.Quantity
	}
	return net
}

// AverageBuyPrice returns the volume-weighted buy price across all executions.
func (p *Position) AverageBuyPrice() decimal.Decimal {
	totalBuy := p.TotalBuyQty()
	totalAmt := p.TotalBuyAmount()
	if totalBuy > 0 && !totalAmt.IsZero() {
		return totalAmt.Div(decimal.NewFromInt(int64(totalBuy)))
	}
	return decimal.Zero
}

// AverageSellPrice returns the volume-weighted sell price across all executions.
func (p *Position) AverageSellPrice() decimal.Decimal {
	totalSell := p.TotalSellQty()
	totalAmt := p.TotalSellAmount()
	if totalSell > 0 && !totalAmt.IsZero() {
		return totalAmt.Div(decimal.NewFromInt(int64(totalSell)))
	}
	return decimal.Zero
}

// RealizedPnL calculates realized profit or loss on closed quantities.
func (p *Position) RealizedPnL() decimal.Decimal {
	closedQty := p.TotalBuyQty()
	if p.TotalSellQty() < closedQty {
		closedQty = p.TotalSellQty()
	}
	if closedQty <= 0 {
		return decimal.Zero
	}
	buyAvg := p.AverageBuyPrice()
	sellAvg := p.AverageSellPrice()
	return sellAvg.Sub(buyAvg).Mul(decimal.NewFromInt(int64(closedQty)))
}

// PositionMetrics aggregates position risk, quantities, and PnL.
type PositionMetrics struct {
	NetQuantity int
	RealizedPnL decimal.Decimal
	TotalMTM    decimal.Decimal
}

// CalculatePositionMetrics summarizes aggregate portfolio metrics across positions.
func CalculatePositionMetrics(positions []Position) PositionMetrics {
	var metrics PositionMetrics
	for _, p := range positions {
		metrics.NetQuantity += p.NetQuantity()
		realized := p.RealizedPnL()
		metrics.RealizedPnL = metrics.RealizedPnL.Add(realized)
		metrics.TotalMTM = metrics.TotalMTM.Add(realized)
	}
	return metrics
}

// Holding represents long-term Demat custody holdings.
type Holding struct {
	InstrumentType     string          `json:"instrumentType"`
	Sector             string          `json:"sector"`
	InstrumentToken    int64           `json:"instrumentToken"`
	CommonScripCode    string          `json:"commonScripCode"`
	InstrumentName     string          `json:"instrumentName"`
	Quantity           decimal.Decimal `json:"quantity"`
	SellableQuantity   decimal.Decimal `json:"sellableQuantity"`
	AveragePrice       decimal.Decimal `json:"averagePrice"`
	HoldingCost        decimal.Decimal `json:"holdingCost"`
	ClosingPrice       decimal.Decimal `json:"closingPrice"`
	MarketValue        decimal.Decimal `json:"mktValue"`
	UnrealisedGainLoss decimal.Decimal `json:"unrealisedGainLoss"`
}

// Limits represents account limits, cash, and collateral.
type Limits struct {
	EntityID        string          `json:"EntityId"`
	BoardLotLimit   decimal.Decimal `json:"BoardLotLimit"`
	CollateralValue decimal.Decimal `json:"CollateralValue"`
	Net             decimal.Decimal `json:"Net"`
	MarginUsed      decimal.Decimal `json:"MarginUsed"`
	AdhocMargin     decimal.Decimal `json:"AdhocMargin"`
	NotionalCash    decimal.Decimal `json:"NotionalCash"`
}

// AvailableCash returns liquid trading cash. If NotionalCash is non-zero, it is returned;
// otherwise liquid cash is derived from Net minus CollateralValue.
func (l *Limits) AvailableCash() decimal.Decimal {
	if !l.NotionalCash.IsZero() {
		return l.NotionalCash
	}
	if !l.Net.IsZero() {
		cash := l.Net.Sub(l.CollateralValue)
		if cash.IsNegative() {
			return decimal.Zero
		}
		return cash
	}
	return decimal.Zero
}

// AvailableMargin returns the total trading margin available (Net limit).
func (l *Limits) AvailableMargin() decimal.Decimal {
	return l.Net
}

// Quote represents a full market quote.
type Quote struct {
	InstrumentToken int64           `json:"instrumentToken"`
	TradingSymbol   string          `json:"tradingSymbol"`
	LastPrice       decimal.Decimal `json:"lastPrice"`
	Volume          int64           `json:"volume"`
	OHLC            QuoteOHLC       `json:"ohlc"`
}

// QuoteOHLC captures Open, High, Low, Close snapshot prices.
type QuoteOHLC struct {
	Open  decimal.Decimal `json:"open"`
	High  decimal.Decimal `json:"high"`
	Low   decimal.Decimal `json:"low"`
	Close decimal.Decimal `json:"close"`
}

// LTPQuote represents lightweight last traded price.
type LTPQuote struct {
	InstrumentToken int64           `json:"instrumentToken"`
	TradingSymbol   string          `json:"tradingSymbol"`
	LastPrice       decimal.Decimal `json:"lastPrice"`
}
