package gokotakneo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DragoHex/gokotakneo"
)

func TestMarket_Execution(t *testing.T) {
	t.Run("Positive: Get Quotes with All Filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/script-details/1.0/quotes/neosymbol/") {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if !strings.HasSuffix(r.URL.Path, "/all") {
				t.Fatalf("expected /all filter suffix: %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "portal-token" {
				t.Errorf("Authorization header missing or incorrect: %q", r.Header.Get("Authorization"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"stat": "Ok",
				"stCode": 200,
				"data": [
					{
						"instrumentToken": 11536,
						"tradingSymbol": "TCS-EQ",
						"lastPrice": "3450.50",
						"volume": 120500,
						"ohlc": {
							"open": "3420.00",
							"high": "3465.00",
							"low": "3410.00",
							"close": "3440.00"
						}
					}
				]
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		quotes, err := client.GetQuotes(context.Background(), []string{"nse_cm|TCS-EQ"}, "all")
		if err != nil {
			t.Fatalf("GetQuotes failed: %v", err)
		}
		if len(quotes) != 1 {
			t.Fatalf("expected 1 quote, got %d", len(quotes))
		}
		tcs := quotes["TCS-EQ"]
		if !tcs.LastPrice.Equal(gokotakneo.MustDecimal("3450.50")) {
			t.Errorf("lastPrice = %s, want 3450.50", tcs.LastPrice)
		}
	})

	t.Run("Positive: Get LTP", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/ltp") {
				t.Fatalf("expected /ltp filter suffix: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"stat": "Ok",
				"stCode": 200,
				"data": [
					{
						"instrumentToken": 11536,
						"tradingSymbol": "TCS-EQ",
						"lastPrice": "3450.50"
					}
				]
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		ltpMap, err := client.GetLTP(context.Background(), []string{"nse_cm|TCS-EQ"})
		if err != nil {
			t.Fatalf("GetLTP failed: %v", err)
		}
		if !ltpMap["TCS-EQ"].LastPrice.Equal(gokotakneo.MustDecimal("3450.50")) {
			t.Errorf("ltp = %s, want 3450.50", ltpMap["TCS-EQ"].LastPrice)
		}
	})
}
