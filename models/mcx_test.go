package models

import (
	"testing"
)

func TestMCXContractUnits_Positive(t *testing.T) {
	tests := []struct {
		symbol   string
		wantUnit int
	}{
		{"CRUDEOIL17NOV265000PE", 100},
		{"CRUDEOIL26NOVFUT", 100},
		{"CRUDEOILM17NOV265000PE", 10},
		{"CRUDEOILM26NOVFUT", 10},
		{"NATURALGAS26NOVFUT", 1250},
		{"NATGASMINI26NOVFUT", 250},
		{"GOLD05DEC26FUT", 100},
		{"GOLDM05DEC26FUT", 10},
		{"GOLDGUINEA26NOVFUT", 8},
		{"GOLDPETAL26NOVFUT", 1},
		{"SILVER05DEC26FUT", 30},
		{"SILVERM05DEC26FUT", 5},
		{"SILVERMIC05DEC26FUT", 1},
		{"COPPER26NOVFUT", 2500},
		{"ZINC26NOVFUT", 5000},
		{"ZINCMINI26NOVFUT", 1000},
		{"ALUMINIUM26NOVFUT", 5000},
		{"ALUMINI26NOVFUT", 1000},
		{"LEAD26NOVFUT", 5000},
		{"LEADMINI26NOVFUT", 1000},
	}

	for _, tt := range tests {
		got := MCXContractUnits(tt.symbol)
		if got != tt.wantUnit {
			t.Errorf("MCXContractUnits(%q) = %d, want %d", tt.symbol, got, tt.wantUnit)
		}
	}
}

func TestNormalizeMCXQuantity_Positive(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		exchange string
		qty      int
		want     int
	}{
		{"Crude oil 1 lot", "CRUDEOIL17NOV265000PE", "MCX", 100, 1},
		{"Crude oil 3 lots", "CRUDEOIL17NOV265000PE", "mcx_fo", 300, 3},
		{"Crude oil mini 2 lots", "CRUDEOILM17NOV265000PE", "MCX", 20, 2},
		{"Natural gas 1 lot", "NATURALGAS26NOVFUT", "MCX", 1250, 1},
		{"Gold 1 lot", "GOLD05DEC26FUT", "MCX", 100, 1},
		{"Gold mini 1 lot", "GOLDM05DEC26FUT", "mcx", 10, 1},
		{"Silver 1 lot", "SILVER05DEC26FUT", "MCX", 30, 1},
		{"Silver mini 1 lot", "SILVERM05DEC26FUT", "MCX", 5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMCXQuantity(tt.symbol, tt.exchange, tt.qty)
			if got != tt.want {
				t.Errorf("NormalizeMCXQuantity(%q, %q, %d) = %d, want %d", tt.symbol, tt.exchange, tt.qty, got, tt.want)
			}
		})
	}
}

func TestNormalizeMCXQuantity_NegativeAndEdge(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		exchange string
		qty      int
		want     int
	}{
		{"Zero quantity", "CRUDEOIL17NOV265000PE", "MCX", 0, 0},
		{"Negative quantity", "CRUDEOIL17NOV265000PE", "MCX", -100, -1},
		{"Unknown commodity", "UNKNOWNCOMM26NOVFUT", "MCX", 50, 50},
		{"Empty symbol", "", "MCX", 100, 100},
		{"Non-MCX NIFTY FO", "NIFTY26NOV24000CE", "nse_fo", 25, 25},
		{"Non-MCX NIFTY FO 50 shares", "NIFTY26NOV24000CE", "NFO", 50, 50},
		{"Non-MCX Equity Cash", "RELIANCE-EQ", "nse_cm", 100, 100},
		{"BSE Equity Cash", "TCS", "bse_cm", 10, 10},
		{"BSE FO", "SENSEX26NOV75000CE", "bse_fo", 20, 20},
		{"MCX case insensitivity MCX_COMM", "CRUDEOIL17NOV265000PE", "MCX_COMM", 100, 1},
		{"MCX case insensitivity lowercase", "CRUDEOIL17NOV265000PE", "mcx", 200, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMCXQuantity(tt.symbol, tt.exchange, tt.qty)
			if got != tt.want {
				t.Errorf("NormalizeMCXQuantity(%q, %q, %d) = %d, want %d", tt.symbol, tt.exchange, tt.qty, got, tt.want)
			}
		})
	}
}

func TestToMCXOrderUnits(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		exchange string
		lots     int
		want     int
	}{
		{"Crude oil 1 lot to units", "CRUDEOIL17NOV265000PE", "MCX", 1, 100},
		{"Crude oil 2 lots to units", "CRUDEOIL17NOV265000PE", "mcx_fo", 2, 200},
		{"Crude oil mini 1 lot to units", "CRUDEOILM17NOV265000PE", "MCX", 1, 10},
		{"Natural gas 2 lots to units", "NATURALGAS26NOVFUT", "MCX", 2, 2500},
		{"Gold 1 lot to units", "GOLD05DEC26FUT", "MCX", 1, 100},
		{"Gold mini 3 lots to units", "GOLDM05DEC26FUT", "MCX", 3, 30},
		{"Non-MCX NIFTY FO preserved", "NIFTY26NOV24000CE", "nse_fo", 25, 25},
		{"Non-MCX Equity Cash preserved", "RELIANCE-EQ", "nse_cm", 100, 100},
		{"Unknown commodity preserved", "UNKNOWNCOMM26NOVFUT", "MCX", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToMCXOrderUnits(tt.symbol, tt.exchange, tt.lots)
			if got != tt.want {
				t.Errorf("ToMCXOrderUnits(%q, %q, %d) = %d, want %d", tt.symbol, tt.exchange, tt.lots, got, tt.want)
			}
		})
	}
}

func TestOrder_NormalizedMCXQuantity(t *testing.T) {
	oMCX := Order{
		ExchangeSegment: "mcx_fo",
		TradingSymbol:   "CRUDEOIL17NOV265000PE",
		Quantity:        200,
		FilledQuantity:  100,
	}

	if got := oMCX.NormalizedMCXQuantity(); got != 2 {
		t.Errorf("NormalizedMCXQuantity() = %d, want 2", got)
	}
	if got := oMCX.NormalizedMCXFilledQuantity(); got != 1 {
		t.Errorf("NormalizedMCXFilledQuantity() = %d, want 1", got)
	}

	oNSE := Order{
		ExchangeSegment: "nse_fo",
		TradingSymbol:   "NIFTY26NOV24000CE",
		Quantity:        50,
		FilledQuantity:  25,
	}

	if got := oNSE.NormalizedMCXQuantity(); got != 50 {
		t.Errorf("Non-MCX NormalizedMCXQuantity() = %d, want 50", got)
	}
	if got := oNSE.NormalizedMCXFilledQuantity(); got != 25 {
		t.Errorf("Non-MCX NormalizedMCXFilledQuantity() = %d, want 25", got)
	}
}
