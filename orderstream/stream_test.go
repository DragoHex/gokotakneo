package orderstream_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DragoHex/gokotakneo/orderstream"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func TestOrderStream_Lifecycle(t *testing.T) {
	t.Run("Positive: Raw String Handshake and Order Fill Demux with Full Fields", func(t *testing.T) {
		var receivedHandshake string
		var handshakeReceived sync.WaitGroup
		handshakeReceived.Add(1)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			// Expect raw handshake string
			_, msg, err := conn.ReadMessage()
			if err == nil {
				receivedHandshake = string(msg)
				handshakeReceived.Done()
			}

			// Send ACK
			conn.WriteJSON(map[string]string{
				"ak":   "ok",
				"type": "cn",
				"msg":  "connected",
			})

			// Push complete mock order execution
			conn.WriteJSON(map[string]interface{}{
				"type": "order",
				"data": map[string]interface{}{
					"nOrdNo":  "260216000308219",
					"exOrdId": "1100000049435826",
					"trdSym":  "ITBEES",
					"sym":     "ITBEES",
					"tok":     "11536",
					"exSeg":   "nse_cm",
					"trnsTp":  "B",
					"prod":    "NRML",
					"prcTp":   "MKT",
					"ordSt":   "complete",
					"qty":     10,
					"fldQty":  10,
					"unFldSz": 0,
					"avgPrc":  "35.88",
					"ordDtTm": "16-Feb-2026 12:29:31",
				},
			})
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		var orderReceived sync.WaitGroup
		orderReceived.Add(1)

		var capturedOrder orderstream.OrderUpdate

		callbacks := orderstream.StreamCallbacks{
			OnOrderUpdate: func(order orderstream.OrderUpdate) {
				capturedOrder = order
				orderReceived.Done()
			},
		}

		client := orderstream.New(wsURL, "my-trade-token", "my-trade-sid", callbacks)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		go client.Serve(ctx)

		handshakeReceived.Wait()
		expectedHandshake := "{type:cn,Authorization:my-trade-token,Sid:my-trade-sid,src:WEB}"
		if receivedHandshake != expectedHandshake {
			t.Errorf("handshake = %q, want %q", receivedHandshake, expectedHandshake)
		}

		orderReceived.Wait()
		if capturedOrder.OrderNumber != "260216000308219" || capturedOrder.Status != "complete" {
			t.Errorf("unexpected order update: %+v", capturedOrder)
		}
		if capturedOrder.ExchangeOrderID != "1100000049435826" {
			t.Errorf("exOrdId = %q, want 1100000049435826", capturedOrder.ExchangeOrderID)
		}
		if capturedOrder.TransactionType != "B" {
			t.Errorf("trnsTp = %q, want B", capturedOrder.TransactionType)
		}
		if capturedOrder.ProductCode != "NRML" {
			t.Errorf("prod = %q, want NRML", capturedOrder.ProductCode)
		}
		if capturedOrder.ExchangeSegment != "nse_cm" {
			t.Errorf("exSeg = %q, want nse_cm", capturedOrder.ExchangeSegment)
		}
		if capturedOrder.InstrumentToken != "11536" {
			t.Errorf("tok = %q, want 11536", capturedOrder.InstrumentToken)
		}
		if capturedOrder.Quantity != 10 || capturedOrder.FilledQuantity != 10 || capturedOrder.UnfilledQuantity != 0 {
			t.Errorf("qty mismatch: qty=%d, fldQty=%d, unFldSz=%d", capturedOrder.Quantity, capturedOrder.FilledQuantity, capturedOrder.UnfilledQuantity)
		}
		if capturedOrder.AveragePrice.String() != "35.88" {
			t.Errorf("avgPrc = %s, want 35.88", capturedOrder.AveragePrice.String())
		}
		if capturedOrder.OrderTimestamp != "16-Feb-2026 12:29:31" {
			t.Errorf("ordDtTm = %q, want 16-Feb-2026 12:29:31", capturedOrder.OrderTimestamp)
		}
	})

	t.Run("Positive: Ping-Pong Keepalive Handling", func(t *testing.T) {
		pongReceived := make(chan string, 1)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			conn.SetPongHandler(func(appData string) error {
				pongReceived <- appData
				return nil
			})

			// Read handshake
			_, _, _ = conn.ReadMessage()

			// Send Ping
			_ = conn.WriteControl(websocket.PingMessage, []byte("server-ping"), time.Now().Add(time.Second))

			// Read messages so pong handler gets invoked
			_, _, _ = conn.ReadMessage()
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		client := orderstream.New(wsURL, "token", "sid", orderstream.StreamCallbacks{})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		go client.Serve(ctx)

		select {
		case data := <-pongReceived:
			if data != "server-ping" {
				t.Errorf("pong appData = %q, want %q", data, "server-ping")
			}
		case <-time.After(time.Second):
			t.Error("timeout waiting for pong response from client ping handler")
		}
	})

	t.Run("Positive: Graceful Close Method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			_, _, _ = conn.ReadMessage()
			// Block until connection is closed by client
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}))
		defer server.Close()

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		connected := make(chan struct{})
		client := orderstream.New(wsURL, "token", "sid", orderstream.StreamCallbacks{
			OnConnect: func() {
				close(connected)
			},
		})

		serveErr := make(chan error, 1)
		go func() {
			serveErr <- client.Serve(context.Background())
		}()

		select {
		case <-connected:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for client to connect")
		}

		// Close client explicitly
		if err := client.Close(); err != nil {
			t.Fatalf("client.Close() failed: %v", err)
		}

		select {
		case <-serveErr:
			// Client exited Serve cleanly
		case <-time.After(time.Second):
			t.Fatal("client.Serve() did not terminate after client.Close()")
		}
	})
}
