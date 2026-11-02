package models

import (
	"strings"
)

type mcxSpec struct {
	prefix string
	units  int
}

// mcxCommoditySpecs is ordered by prefix length descending to ensure
// mini and micro variations (e.g. CRUDEOILM, GOLDM, SILVERMIC) are matched
// before their base underlying names.
var mcxCommoditySpecs = []mcxSpec{
	{"COTTONCANDY", 48},
	{"NATGASMINI", 250},
	{"CRUDEOILM", 10},
	{"CRUDEOIL", 100},
	{"NATURALGAS", 1250},
	{"GOLDGUINEA", 8},
	{"GOLDPETAL", 1},
	{"GOLDM", 10},
	{"GOLD", 100},
	{"SILVERMIC", 1},
	{"SILVERM", 5},
	{"SILVER", 30},
	{"ALUMINIUM", 5000},
	{"ALUMINI", 1000},
	{"ZINCMINI", 1000},
	{"ZINC", 5000},
	{"LEADMINI", 1000},
	{"LEAD", 5000},
	{"COPPER", 2500},
	{"NICKEL", 1500},
	{"COTTON", 25},
	{"MENTHAOIL", 360},
}

// IsMCX returns true if the exchange segment refers to MCX commodities.
func IsMCX(exchange string) bool {
	ex := strings.ToLower(strings.TrimSpace(exchange))
	return strings.HasPrefix(ex, "mcx")
}

// MCXContractUnits returns the unit multiplier per lot for an MCX commodity contract.
// Returns 1 for non-MCX, empty, or unknown commodities.
func MCXContractUnits(symbol string) int {
	clean := strings.ToUpper(strings.TrimSpace(symbol))
	if clean == "" {
		return 1
	}

	for _, spec := range mcxCommoditySpecs {
		if strings.HasPrefix(clean, spec.prefix) {
			return spec.units
		}
	}

	return 1
}

// NormalizeMCXQuantity converts raw commodity units into standard lots for MCX contracts.
// For non-MCX exchanges (NSE, BSE, etc.), it returns quantity untouched.
func NormalizeMCXQuantity(symbol, exchange string, quantity int) int {
	if !IsMCX(exchange) || quantity == 0 {
		return quantity
	}

	units := MCXContractUnits(symbol)
	if units <= 1 {
		return quantity
	}

	return quantity / units
}

// ToMCXOrderUnits converts standard lots into Kotak order units (e.g. 1 lot -> 100 barrels for CRUDEOIL).
// For non-MCX exchanges, it returns lots untouched.
func ToMCXOrderUnits(symbol, exchange string, lots int) int {
	if !IsMCX(exchange) || lots == 0 {
		return lots
	}

	units := MCXContractUnits(symbol)
	if units <= 1 {
		return lots
	}

	return lots * units
}
