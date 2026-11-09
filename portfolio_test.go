package gokotakneo_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/DragoHex/gokotakneo"
	"github.com/shopspring/decimal"
)

func TestPortfolio_Execution(t *testing.T) {
	t.Run("Positive: Get Positions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/user/positions" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"stat": "Ok",
				"stCode": 200,
				"data": [
					{
						"actId": "ACT123",
						"prod": "CNC",
						"exSeg": "nse_cm",
						"trdSym": "AXISBANK-EQ",
						"sym": "AXISBANK",
						"qty": "9",
						"buyAmt": "5862.90",
						"sellAmt": "0.00",
						"flBuyQty": "9",
						"flSellQty": "0",
						"brdLtQty": 1,
						"lotSz": "1",
						"stkPrc": "0.00",
						"hsUpTm": "2022/06/21 15:11:02"
					}
				]
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		positions, err := client.GetPositions(context.Background())
		if err != nil {
			t.Fatalf("GetPositions failed: %v", err)
		}
		if len(positions) != 1 {
			t.Fatalf("expected 1 position, got %d", len(positions))
		}
		if positions[0].TradingSymbol != "AXISBANK-EQ" || positions[0].Quantity != 9 {
			t.Errorf("unexpected position: %+v", positions[0])
		}
		if !positions[0].BuyAmount.Equal(gokotakneo.MustDecimal("5862.90")) {
			t.Errorf("buyAmt = %s, want 5862.90", positions[0].BuyAmount)
		}
	})

	t.Run("Positive: Get Holdings", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/portfolio/v1/holdings" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": [
					{
						"instrumentType": "Equity",
						"sector": "Others",
						"instrumentToken": 18974,
						"commonScripCode": "NIPPON_ETF_NIFTYIT",
						"instrumentName": "Nippon India ETF Nifty IT",
						"quantity": 45,
						"averagePrice": 38.7162,
						"holdingCost": 1742.2293,
						"closingPrice": 39.36,
						"mktValue": 1771.2,
						"unrealisedGainLoss": 28.9707
					}
				]
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		holdings, err := client.GetHoldings(context.Background())
		if err != nil {
			t.Fatalf("GetHoldings failed: %v", err)
		}
		if len(holdings) != 1 {
			t.Fatalf("expected 1 holding, got %d", len(holdings))
		}
		if !holdings[0].Quantity.Equal(decimal.NewFromInt(45)) || holdings[0].CommonScripCode != "NIPPON_ETF_NIFTYIT" {
			t.Errorf("unexpected holding: %+v", holdings[0])
		}
	})

	t.Run("Positive: Get Holdings with Fractional and Decimal Quantities", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/portfolio/v1/holdings" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": [
					{
						"instrumentType": "Equity",
						"sector": "Others",
						"instrumentToken": 2965,
						"commonScripCode": "BENCHMARK_MUTUAL_FUN",
						"instrumentName": "Nippon India ETF Nifty 1D Rate Liquid BeES",
						"quantity": 0.002,
						"averagePrice": 0,
						"holdingCost": 0,
						"closingPrice": 1000,
						"mktValue": 2,
						"sellableQuantity": 0
					},
					{
						"instrumentType": "mutual fund",
						"sector": "Equity",
						"instrumentToken": 0,
						"commonScripCode": "PP001ZG-GR",
						"instrumentName": "Parag Parikh Flexi Cap Fund - Growth|INF879O01027",
						"quantity": 15456.843,
						"averagePrice": 88.8675,
						"holdingCost": 1373610.995,
						"closingPrice": 93.9483,
						"mktValue": 1452144.123,
						"sellableQuantity": 0
					},
					{
						"instrumentType": "COMM",
						"sector": "Others",
						"instrumentToken": 0,
						"commonScripCode": "OPTFUTCRUDEOIL",
						"instrumentName": "OPTFUTCRUDEOIL",
						"quantity": -100,
						"averagePrice": 38.6629,
						"holdingCost": -3866.29,
						"closingPrice": 39.4,
						"mktValue": -3940,
						"sellableQuantity": 0
					}
				]
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		holdings, err := client.GetHoldings(context.Background())
		if err != nil {
			t.Fatalf("GetHoldings failed: %v", err)
		}
		if len(holdings) != 3 {
			t.Fatalf("expected 3 holdings, got %d", len(holdings))
		}
		if !holdings[0].Quantity.Equal(decimal.NewFromFloat(0.002)) {
			t.Errorf("holdings[0].Quantity = %s, want 0.002", holdings[0].Quantity)
		}
		if !holdings[1].Quantity.Equal(decimal.NewFromFloat(15456.843)) {
			t.Errorf("holdings[1].Quantity = %s, want 15456.843", holdings[1].Quantity)
		}
		if !holdings[2].Quantity.Equal(decimal.NewFromInt(-100)) {
			t.Errorf("holdings[2].Quantity = %s, want -100", holdings[2].Quantity)
		}
	})

	t.Run("Positive: Get Limits", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/user/limits" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			jData := values.Get("jData")
			if !strings.Contains(jData, `"seg":"ALL"`) {
				t.Errorf("missing seg:ALL in limits request: %s", jData)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"Category": "CLIENT_MTF",
				"EntityId": "account-123",
				"BoardLotLimit": "5000",
				"CollateralValue": "10197.48",
				"Net": "10157.08",
				"MarginUsed": "40.4"
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		limits, err := client.GetLimits(context.Background(), "ALL", "ALL", "ALL")
		if err != nil {
			t.Fatalf("GetLimits failed: %v", err)
		}
		if limits.EntityID != "account-123" {
			t.Errorf("EntityID = %q, want 'account-123'", limits.EntityID)
		}
		if !limits.Net.Equal(gokotakneo.MustDecimal("10157.08")) {
			t.Errorf("Net = %s, want 10157.08", limits.Net)
		}
	})

	t.Run("Positive: GetPositions empty on 5203 No Data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"stCode":5203,"errMsg":"No Data","desc":"data not found","stat":"Not_Ok"}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		positions, err := client.GetPositions(context.Background())
		if err != nil {
			t.Fatalf("expected nil error on 5203, got: %v", err)
		}
		if len(positions) != 0 {
			t.Errorf("expected 0 positions, got %d", len(positions))
		}
	})

	t.Run("Positive: GetHoldings empty on 424 No holdings found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(424)
			w.Write([]byte(`{"error":[{"code":424,"message":"No holdings found for this client V45IC"}]}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		holdings, err := client.GetHoldings(context.Background())
		if err != nil {
			t.Fatalf("expected nil error on 424 no holdings, got: %v", err)
		}
		if len(holdings) != 0 {
			t.Errorf("expected 0 holdings, got %d", len(holdings))
		}
	})
}
