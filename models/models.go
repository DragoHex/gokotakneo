package models

import (
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
	FilledBuyQty    int             `json:"flBuyQty,string"`
	FilledSellQty   int             `json:"flSellQty,string"`
	BoardLotQty     int             `json:"brdLtQty"`
	SquareOffFlag   string          `json:"sqrFlg"`
	LotSize         int             `json:"lotSz,string"`
	StrikePrice     decimal.Decimal `json:"stkPrc"`
	UpdateTime      BrokerTime      `json:"hsUpTm"`
}

// Holding represents long-term Demat custody holdings.
type Holding struct {
	InstrumentType     string          `json:"instrumentType"`
	Sector             string          `json:"sector"`
	InstrumentToken    int64           `json:"instrumentToken"`
	CommonScripCode    string          `json:"commonScripCode"`
	InstrumentName     string          `json:"instrumentName"`
	Quantity           int             `json:"quantity"`
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
