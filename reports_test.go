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

func TestReports_Execution(t *testing.T) {
	t.Run("Positive: Get Order Book", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/user/orders" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"stat": "Ok",
				"stCode": 200,
				"data": [
					{
						"nOrdNo": "250720000007242",
						"trdSym": "INFY-EQ",
						"exSeg": "nse_cm",
						"trnsTp": "B",
						"prod": "CNC",
						"prcTp": "MKT",
						"ordSt": "complete",
						"qty": 10,
						"fldQty": 10,
						"avgPrc": 1450.50,
						"ordDtTm": "2023/10/16 10:15:30"
					}
				]
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		orders, err := client.GetOrders(context.Background())
		if err != nil {
			t.Fatalf("GetOrders failed: %v", err)
		}
		if len(orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(orders))
		}
		if orders[0].OrderNumber != "250720000007242" || orders[0].Status != "complete" {
			t.Errorf("unexpected order: %+v", orders[0])
		}
		if !orders[0].AveragePrice.Equal(gokotakneo.MustDecimal("1450.50")) {
			t.Errorf("avg price = %s, want 1450.50", orders[0].AveragePrice)
		}
	})

	t.Run("Positive: Get Order by ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/user/orders/250720000007242" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"stat": "Ok",
				"stCode": 200,
				"data": {
					"nOrdNo": "250720000007242",
					"trdSym": "INFY-EQ",
					"ordSt": "open"
				}
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		order, err := client.GetOrderByID(context.Background(), "250720000007242")
		if err != nil {
			t.Fatalf("GetOrderByID failed: %v", err)
		}
		if order.OrderNumber != "250720000007242" || order.Status != "open" {
			t.Errorf("unexpected order: %+v", order)
		}
	})

	t.Run("Positive: Get Order History", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/user/order/history" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			jData := values.Get("jData")
			if !strings.Contains(jData, `"nOrdNo":"250720000007242"`) {
				t.Errorf("missing nOrdNo in jData: %s", jData)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"stat": "Ok",
				"stCode": 200,
				"data": [
					{"nOrdNo": "250720000007242", "ordSt": "open"},
					{"nOrdNo": "250720000007242", "ordSt": "complete"}
				]
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		history, err := client.GetOrderHistory(context.Background(), "250720000007242")
		if err != nil {
			t.Fatalf("GetOrderHistory failed: %v", err)
		}
		if len(history) != 2 {
			t.Fatalf("expected 2 history entries, got %d", len(history))
		}
	})

	t.Run("Positive: Get Trade Book", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/user/trades" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"stat": "Ok",
				"stCode": 200,
				"data": [
					{
						"trdId": "TRD12345",
						"nOrdNo": "250720000007242",
						"trdSym": "INFY-EQ",
						"qty": 10,
						"prc": 1450.50
					}
				]
			}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		trades, err := client.GetTrades(context.Background())
		if err != nil {
			t.Fatalf("GetTrades failed: %v", err)
		}
		if len(trades) != 1 {
			t.Fatalf("expected 1 trade, got %d", len(trades))
		}
		if trades[0].TradeID != "TRD12345" {
			t.Errorf("tradeID = %q, want TRD12345", trades[0].TradeID)
		}
	})
}
