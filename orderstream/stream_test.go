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
	t.Run("Positive: Raw String Handshake and Order Fill Demux", func(t *testing.T) {
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

			// Push mock order execution
			conn.WriteJSON(map[string]interface{}{
				"type": "order",
				"data": map[string]interface{}{
					"nOrdNo":  "250720000007242",
					"trdSym":  "INFY-EQ",
					"ordSt":   "complete",
					"fldQty":  10,
					"avgPrc":  "1450.50",
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
		if capturedOrder.OrderNumber != "250720000007242" || capturedOrder.Status != "complete" {
			t.Errorf("unexpected order update: %+v", capturedOrder)
		}
	})
}
