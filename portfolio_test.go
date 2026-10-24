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
		if holdings[0].Quantity != 45 || holdings[0].CommonScripCode != "NIPPON_ETF_NIFTYIT" {
			t.Errorf("unexpected holding: %+v", holdings[0])
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
}
