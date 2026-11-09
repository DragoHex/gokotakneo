package models_test

import (
	"encoding/json"
	"testing"

	"github.com/DragoHex/gokotakneo/models"
	"github.com/shopspring/decimal"
)

func TestLimits_AvailableCashAndMargin(t *testing.T) {
	t.Run("Positive: NotionalCash is populated", func(t *testing.T) {
		l := models.Limits{
			NotionalCash:    decimal.NewFromFloat(500.50),
			Net:             decimal.NewFromFloat(1500.00),
			CollateralValue: decimal.NewFromFloat(200.00),
		}
		if !l.AvailableCash().Equal(decimal.NewFromFloat(500.50)) {
			t.Errorf("AvailableCash = %s, want 500.50", l.AvailableCash())
		}
		if !l.AvailableMargin().Equal(decimal.NewFromFloat(1500.00)) {
			t.Errorf("AvailableMargin = %s, want 1500.00", l.AvailableMargin())
		}
	})

	t.Run("Positive: NotionalCash is zero, derived from Net minus CollateralValue", func(t *testing.T) {
		l := models.Limits{
			NotionalCash:    decimal.Zero,
			Net:             decimal.NewFromInt(970),
			CollateralValue: decimal.Zero,
		}
		if !l.AvailableCash().Equal(decimal.NewFromInt(970)) {
			t.Errorf("AvailableCash = %s, want 970", l.AvailableCash())
		}
		if !l.AvailableMargin().Equal(decimal.NewFromInt(970)) {
			t.Errorf("AvailableMargin = %s, want 970", l.AvailableMargin())
		}
	})

	t.Run("Positive: Collateral deducted from Net for cash", func(t *testing.T) {
		l := models.Limits{
			NotionalCash:    decimal.Zero,
			Net:             decimal.NewFromInt(1000),
			CollateralValue: decimal.NewFromInt(400),
		}
		if !l.AvailableCash().Equal(decimal.NewFromInt(600)) {
			t.Errorf("AvailableCash = %s, want 600", l.AvailableCash())
		}
	})

	t.Run("Edge Case: Collateral exceeds Net floors at zero", func(t *testing.T) {
		l := models.Limits{
			NotionalCash:    decimal.Zero,
			Net:             decimal.NewFromInt(300),
			CollateralValue: decimal.NewFromInt(500),
		}
		if !l.AvailableCash().Equal(decimal.Zero) {
			t.Errorf("AvailableCash = %s, want 0", l.AvailableCash())
		}
	})
}

func TestOrder_Normalization(t *testing.T) {
	t.Run("TransactionType normalization", func(t *testing.T) {
		cases := []struct {
			input string
			want  string
		}{
			{"B", "BUY"},
			{"b", "BUY"},
			{"BUY", "BUY"},
			{"buy", "BUY"},
			{"S", "SELL"},
			{"s", "SELL"},
			{"SELL", "SELL"},
			{"", "BUY"},
		}
		for _, tc := range cases {
			o := models.Order{TransactionType: tc.input}
			if got := o.NormalizedTransactionType(); got != tc.want {
				t.Errorf("NormalizedTransactionType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		}
	})

	t.Run("OrderType normalization", func(t *testing.T) {
		cases := []struct {
			input string
			want  string
		}{
			{"L", "LIMIT"},
			{"LIMIT", "LIMIT"},
			{"MKT", "MARKET"},
			{"MARKET", "MARKET"},
			{"SL", "SL"},
			{"SL-M", "SL-M"},
			{"SLM", "SL-M"},
			{"", "MARKET"},
		}
		for _, tc := range cases {
			o := models.Order{PriceType: tc.input}
			if got := o.NormalizedOrderType(); got != tc.want {
				t.Errorf("NormalizedOrderType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		}
	})

	t.Run("Product normalization", func(t *testing.T) {
		cases := []struct {
			input string
			want  string
		}{
			{"CNC", "CNC"},
			{"cnc", "CNC"},
			{"NRML", "NRML"},
			{"MARGIN", "NRML"},
			{"MIS", "MIS"},
			{"INTRADAY", "MIS"},
		}
		for _, tc := range cases {
			o := models.Order{ProductCode: tc.input}
			if got := o.NormalizedProduct(); got != tc.want {
				t.Errorf("NormalizedProduct(%q) = %q, want %q", tc.input, got, tc.want)
			}
		}
	})

	t.Run("Exchange normalization", func(t *testing.T) {
		cases := []struct {
			input string
			want  string
		}{
			{"nse_fo", "NFO"},
			{"nfo", "NFO"},
			{"bse_fo", "BFO"},
			{"bfo", "BFO"},
			{"mcx_fo", "MCX"},
			{"mcx", "MCX"},
			{"bse_cm", "BSE"},
			{"bse", "BSE"},
			{"nse_cm", "NSE"},
			{"nse", "NSE"},
		}
		for _, tc := range cases {
			o := models.Order{ExchangeSegment: tc.input}
			if got := o.NormalizedExchange(); got != tc.want {
				t.Errorf("NormalizedExchange(%q) = %q, want %q", tc.input, got, tc.want)
			}
		}
	})

	t.Run("SymbolName fallback", func(t *testing.T) {
		o1 := models.Order{TradingSymbol: "RELIANCE-EQ", Symbol: "RELIANCE"}
		if o1.SymbolName() != "RELIANCE-EQ" {
			t.Errorf("SymbolName() = %q, want 'RELIANCE-EQ'", o1.SymbolName())
		}

		o2 := models.Order{TradingSymbol: "", Symbol: "RELIANCE"}
		if o2.SymbolName() != "RELIANCE" {
			t.Errorf("SymbolName() = %q, want 'RELIANCE'", o2.SymbolName())
		}
	})

	t.Run("IsTerminal check", func(t *testing.T) {
		if !(&models.Order{Status: "COMPLETE"}).IsTerminal() {
			t.Error("COMPLETE should be terminal")
		}
		if !(&models.Order{Status: "REJECTED"}).IsTerminal() {
			t.Error("REJECTED should be terminal")
		}
		if !(&models.Order{Status: "CANCELLED"}).IsTerminal() {
			t.Error("CANCELLED should be terminal")
		}
		if (&models.Order{Status: "OPEN"}).IsTerminal() {
			t.Error("OPEN should not be terminal")
		}
	})
}

func TestPosition_MetricsAndPnL(t *testing.T) {
	t.Run("RealizedPnL for closed trade", func(t *testing.T) {
		// Bought 100 @ 6.50 (Amt = 650.00), Sold 100 @ 6.20 (Amt = 620.00)
		p := models.Position{
			TradingSymbol: "CRUDEOIL17NOV265000PE",
			Quantity:      0,
			FilledBuyQty:  100,
			BuyAmount:     decimal.NewFromFloat(650.00),
			FilledSellQty: 100,
			SellAmount:    decimal.NewFromFloat(620.00),
		}

		wantPnL := decimal.NewFromFloat(-30.00)
		if !p.RealizedPnL().Equal(wantPnL) {
			t.Errorf("RealizedPnL() = %s, want -30.00", p.RealizedPnL())
		}
		if p.NetQuantity() != 0 {
			t.Errorf("NetQuantity() = %d, want 0", p.NetQuantity())
		}
		if !p.AverageBuyPrice().Equal(decimal.NewFromFloat(6.50)) {
			t.Errorf("AverageBuyPrice() = %s, want 6.50", p.AverageBuyPrice())
		}
		if !p.AverageSellPrice().Equal(decimal.NewFromFloat(6.20)) {
			t.Errorf("AverageSellPrice() = %s, want 6.20", p.AverageSellPrice())
		}
	})

	t.Run("CalculatePositionMetrics aggregate", func(t *testing.T) {
		positions := []models.Position{
			{
				TradingSymbol: "CRUDEOIL17NOV265000PE",
				Quantity:      0,
				FilledBuyQty:  100,
				BuyAmount:     decimal.NewFromFloat(650.00),
				FilledSellQty: 100,
				SellAmount:    decimal.NewFromFloat(620.00),
			},
			{
				TradingSymbol: "GOLDM27NOV26160000CE",
				Quantity:      10,
				FilledBuyQty:  10,
				BuyAmount:     decimal.NewFromFloat(1000.00),
				FilledSellQty: 0,
				SellAmount:    decimal.Zero,
			},
		}

		metrics := models.CalculatePositionMetrics(positions)
		if metrics.NetQuantity != 10 {
			t.Errorf("metrics.NetQuantity = %d, want 10", metrics.NetQuantity)
		}
		if !metrics.RealizedPnL.Equal(decimal.NewFromFloat(-30.00)) {
			t.Errorf("metrics.RealizedPnL = %s, want -30.00", metrics.RealizedPnL)
		}
		// TotalMTM is sum of RealizedPnL when no unclosed MTM feed is supplied
		if !metrics.TotalMTM.Equal(decimal.NewFromFloat(-30.00)) {
			t.Errorf("metrics.TotalMTM = %s, want -30.00", metrics.TotalMTM)
		}
	})

	t.Run("Positive: UnmarshalLivePayload with CarriedForward and Day executions", func(t *testing.T) {
		overnightShortJSON := `{
			"actId": "V4KI4",
			"brdLtQty": 100,
			"cfBuyAmt": "0.00",
			"cfSellAmt": "2640.00",
			"cfBuyQty": "0",
			"cfSellQty": "100",
			"exSeg": "mcx_fo",
			"buyAmt": "0.00",
			"sellAmt": "0.00",
			"flBuyQty": "0",
			"flSellQty": "0",
			"prod": "NRML",
			"trdSym": "CRUDEOIL17NOV267300PE",
			"lotSz": "100"
		}`

		var p1 models.Position
		if err := json.Unmarshal([]byte(overnightShortJSON), &p1); err != nil {
			t.Fatalf("unmarshal p1: %v", err)
		}
		if p1.Quantity != -100 {
			t.Errorf("p1.Quantity = %d, want -100", p1.Quantity)
		}
		if p1.NetQuantity() != -100 {
			t.Errorf("p1.NetQuantity() = %d, want -100", p1.NetQuantity())
		}
		if !p1.AverageSellPrice().Equal(decimal.NewFromFloat(26.40)) {
			t.Errorf("p1.AverageSellPrice() = %s, want 26.40", p1.AverageSellPrice())
		}

		dayShortJSON := `{
			"actId": "V4KI4",
			"brdLtQty": 65,
			"cfBuyAmt": "0.00",
			"cfSellAmt": "0.00",
			"cfBuyQty": "0",
			"cfSellQty": "0",
			"exSeg": "nse_fo",
			"buyAmt": "0.00",
			"sellAmt": "4143.75",
			"flBuyQty": "0",
			"flSellQty": "325",
			"prod": "NRML",
			"trdSym": "NIFTY26N0924600PE",
			"lotSz": "65"
		}`

		var p2 models.Position
		if err := json.Unmarshal([]byte(dayShortJSON), &p2); err != nil {
			t.Fatalf("unmarshal p2: %v", err)
		}
		if p2.Quantity != -325 {
			t.Errorf("p2.Quantity = %d, want -325", p2.Quantity)
		}
		if p2.NetQuantity() != -325 {
			t.Errorf("p2.NetQuantity() = %d, want -325", p2.NetQuantity())
		}
		if !p2.AverageSellPrice().Equal(decimal.NewFromFloat(12.75)) {
			t.Errorf("p2.AverageSellPrice() = %s, want 12.75", p2.AverageSellPrice())
		}

		closedTradeJSON := `{
			"actId": "V4KI4",
			"brdLtQty": 65,
			"cfBuyAmt": "0.00",
			"cfSellAmt": "0.00",
			"cfBuyQty": "0",
			"cfSellQty": "0",
			"exSeg": "nse_fo",
			"buyAmt": "1251.25",
			"sellAmt": "5053.75",
			"flBuyQty": "325",
			"flSellQty": "325",
			"prod": "NRML",
			"trdSym": "NIFTY26N0925300CE",
			"lotSz": "65"
		}`

		var p3 models.Position
		if err := json.Unmarshal([]byte(closedTradeJSON), &p3); err != nil {
			t.Fatalf("unmarshal p3: %v", err)
		}
		if p3.Quantity != 0 {
			t.Errorf("p3.Quantity = %d, want 0", p3.Quantity)
		}
		if p3.NetQuantity() != 0 {
			t.Errorf("p3.NetQuantity() = %d, want 0", p3.NetQuantity())
		}
		if !p3.AverageBuyPrice().Equal(decimal.NewFromFloat(3.85)) {
			t.Errorf("p3.AverageBuyPrice() = %s, want 3.85", p3.AverageBuyPrice())
		}
		if !p3.AverageSellPrice().Equal(decimal.NewFromFloat(15.55)) {
			t.Errorf("p3.AverageSellPrice() = %s, want 15.55", p3.AverageSellPrice())
		}
		if !p3.RealizedPnL().Equal(decimal.NewFromFloat(3802.50)) {
			t.Errorf("p3.RealizedPnL() = %s, want 3802.50", p3.RealizedPnL())
		}
	})
}
