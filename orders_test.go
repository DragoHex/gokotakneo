package gokotakneo_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/DragoHex/gokotakneo"
	"github.com/DragoHex/gokotakneo/models"
)

func TestOrders_Execution(t *testing.T) {
	t.Run("Positive: Place Market Buy Order", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/order/rule/ms/place" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			jData := values.Get("jData")

			if !strings.Contains(jData, `"ts":"INFY-EQ"`) || !strings.Contains(jData, `"pt":"MKT"`) || !strings.Contains(jData, `"tt":"B"`) {
				t.Errorf("jData missing expected order fields: %s", jData)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"stat": "Ok", "stCode": 200, "nOrdNo": "250720000007242"}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		resp, err := client.PlaceOrder(context.Background(), models.OrderParams{
			Am: "NO",
			Dq: "0",
			Es: "nse_cm",
			Mp: "0",
			Pc: "CNC",
			Pf: "N",
			Pr: "0",
			Pt: "MKT",
			Qt: "10",
			Rt: "DAY",
			Tp: "0",
			Ts: "INFY-EQ",
			Tt: "B",
		})
		if err != nil {
			t.Fatalf("PlaceOrder failed: %v", err)
		}
		if resp.OrderNumber != "250720000007242" || resp.Stat != "Ok" {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("Positive: Modify Open Order", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/order/vr/modify" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			jData := values.Get("jData")

			if !strings.Contains(jData, `"no":"250720000007242"`) || !strings.Contains(jData, `"pr":"1500"`) {
				t.Errorf("jData missing modified fields: %s", jData)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"stat": "Ok", "stCode": 200, "nOrdNo": "250720000007242"}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		resp, err := client.ModifyOrder(context.Background(), models.OrderModifyParams{
			OrderNo: "250720000007242",
			Pr:      "1500",
			Qt:      "20",
		})
		if err != nil {
			t.Fatalf("ModifyOrder failed: %v", err)
		}
		if resp.OrderNumber != "250720000007242" {
			t.Errorf("unexpected modified order number: %q", resp.OrderNumber)
		}
	})

	t.Run("Positive: Cancel Order", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/quick/order/cancel" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			jData := values.Get("jData")

			if !strings.Contains(jData, `"on":"250720000007242"`) {
				t.Errorf("jData missing cancel order number: %s", jData)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"stat": "Ok", "stCode": 200, "nOrdNo": "250720000007242"}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		resp, err := client.CancelOrder(context.Background(), "250720000007242", false, "")
		if err != nil {
			t.Fatalf("CancelOrder failed: %v", err)
		}
		if resp.OrderNumber != "250720000007242" {
			t.Errorf("unexpected cancel response: %+v", resp)
		}
	})

	t.Run("Negative: Insufficient Balance Rejection (1004)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"stat": "Not_Ok", "stCode": 1004, "emsg": "Insufficient balance."}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		_, err := client.PlaceOrder(context.Background(), models.OrderParams{
			Ts: "RELIANCE-EQ",
			Qt: "1000",
		})
		if err == nil {
			t.Fatalf("expected error from RMS rejection, got nil")
		}
		if !errors.Is(err, gokotakneo.ErrInsufficientFunds) {
			t.Errorf("expected ErrInsufficientFunds, got: %v", err)
		}
	})

	t.Run("Edge Case: Empty Order No in Cancel", func(t *testing.T) {
		client := gokotakneo.New("portal-token")
		_, err := client.CancelOrder(context.Background(), "", false, "")
		if err == nil {
			t.Fatalf("expected error on empty order number, got nil")
		}
		if !errors.Is(err, gokotakneo.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got: %v", err)
		}
	})
}
