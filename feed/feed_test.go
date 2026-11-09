package feed_test

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DragoHex/gokotakneo/feed"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

var upgrader = websocket.Upgrader{}

func TestNativeBatchBinaryParser(t *testing.T) {
	t.Run("Positive: Single LTP Packet with Divider Scaling", func(t *testing.T) {
		// Mock LTP Packet:
		// [0..1] Length = 8
		// [2..3] MessageCode = 1 (LTP)
		// [4..7] InstrumentToken = 11536 (little endian uint32)
		// [8..11] Scaled Price = 345050 (little endian int32) -> 3450.50 with divider 100
		buf := make([]byte, 12)
		binary.LittleEndian.PutUint16(buf[0:2], 8)
		binary.LittleEndian.PutUint16(buf[2:4], 1)
		binary.LittleEndian.PutUint32(buf[4:8], 11536)
		binary.LittleEndian.PutUint32(buf[8:12], 345050)

		parser := feed.NewParser()
		parser.SetDivider("nse_cm", 100.0)

		ticks, err := parser.ParseBatch(buf, "nse_cm")
		if err != nil {
			t.Fatalf("ParseBatch failed: %v", err)
		}
		if len(ticks) != 1 {
			t.Fatalf("expected 1 tick, got %d", len(ticks))
		}
		tick := ticks[0]
		if tick.InstrumentToken != 11536 {
			t.Errorf("token = %d, want 11536", tick.InstrumentToken)
		}
		expectedPrice := decimal.NewFromFloat(3450.50)
		if !tick.LastPrice.Equal(expectedPrice) {
			t.Errorf("lastPrice = %s, want %s", tick.LastPrice, expectedPrice)
		}
	})

	t.Run("Positive: Multi-Packet Batch De-framing", func(t *testing.T) {
		// Construct two concatenated packets
		pkt1 := make([]byte, 12)
		binary.LittleEndian.PutUint16(pkt1[0:2], 8)
		binary.LittleEndian.PutUint16(pkt1[2:4], 1)
		binary.LittleEndian.PutUint32(pkt1[4:8], 1001)
		binary.LittleEndian.PutUint32(pkt1[8:12], 10000) // 100.00

		pkt2 := make([]byte, 12)
		binary.LittleEndian.PutUint16(pkt2[0:2], 8)
		binary.LittleEndian.PutUint16(pkt2[2:4], 1)
		binary.LittleEndian.PutUint32(pkt2[4:8], 1002)
		binary.LittleEndian.PutUint32(pkt2[8:12], 25050) // 250.50

		batch := append(pkt1, pkt2...)

		parser := feed.NewParser()
		parser.SetDivider("nse_cm", 100.0)

		ticks, err := parser.ParseBatch(batch, "nse_cm")
		if err != nil {
			t.Fatalf("ParseBatch failed: %v", err)
		}
		if len(ticks) != 2 {
			t.Fatalf("expected 2 ticks, got %d", len(ticks))
		}
		if ticks[0].InstrumentToken != 1001 || ticks[1].InstrumentToken != 1002 {
			t.Errorf("unexpected tokens: %v", ticks)
		}
	})

	t.Run("Negative: Truncated Buffer", func(t *testing.T) {
		parser := feed.NewParser()
		_, err := parser.ParseBatch([]byte{0x08, 0x00}, "nse_cm")
		if err == nil {
			t.Fatalf("expected error on truncated packet, got nil")
		}
	})

	t.Run("Edge Case: Zero Divider Fallback", func(t *testing.T) {
		buf := make([]byte, 12)
		binary.LittleEndian.PutUint16(buf[0:2], 8)
		binary.LittleEndian.PutUint16(buf[2:4], 1)
		binary.LittleEndian.PutUint32(buf[4:8], 11536)
		binary.LittleEndian.PutUint32(buf[8:12], 10000)

		parser := feed.NewParser()
		// No divider set -> should fallback safely to 100.0 without divide by zero
		ticks, err := parser.ParseBatch(buf, "unknown_seg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ticks[0].LastPrice.Equal(decimal.NewFromFloat(100.00)) {
			t.Errorf("lastPrice = %s, want 100.00", ticks[0].LastPrice)
		}
	})
}

func TestFeed_WithHTTPSProxy(t *testing.T) {
	// Target WebSocket server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Read auth handshake
		var authPayload map[string]interface{}
		_ = conn.ReadJSON(&authPayload)

		// Send sample binary packet
		buf := make([]byte, 12)
		binary.LittleEndian.PutUint16(buf[0:2], 8)
		binary.LittleEndian.PutUint16(buf[2:4], 1)
		binary.LittleEndian.PutUint32(buf[4:8], 11536)
		binary.LittleEndian.PutUint32(buf[8:12], 345050)
		_ = conn.WriteMessage(websocket.BinaryMessage, buf)

		// Keep alive briefly
		time.Sleep(100 * time.Millisecond)
	}))
	defer targetServer.Close()

	var proxyConnectCount atomic.Int32
	// Mock HTTPS forward proxy
	proxyServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			proxyConnectCount.Add(1)
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijacking not supported", http.StatusInternalServerError)
				return
			}
			clientConn, bufrw, err := hj.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer clientConn.Close()

			targetConn, err := net.DialTimeout("tcp", r.Host, 5*time.Second)
			if err != nil {
				bufrw.WriteString("HTTP/1.1 502 Bad Gateway\r\n\r\n")
				bufrw.Flush()
				return
			}
			defer targetConn.Close()

			bufrw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
			bufrw.Flush()

			errc := make(chan error, 2)
			go func() {
				_, err := io.Copy(targetConn, bufrw)
				errc <- err
			}()
			go func() {
				_, err := io.Copy(clientConn, targetConn)
				errc <- err
			}()
			<-errc
			return
		}
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}))
	defer proxyServer.Close()

	wsURL := "ws" + strings.TrimPrefix(targetServer.URL, "http")
	proxyURL := "https://proxyuser:proxypass@" + strings.TrimPrefix(proxyServer.URL, "https://")

	var tickReceived sync.WaitGroup
	tickReceived.Add(1)
	var capturedTick feed.Tick

	callbacks := feed.FeedCallbacks{
		OnTick: func(tick feed.Tick) {
			capturedTick = tick
			tickReceived.Done()
		},
	}

	client := feed.NewClient(wsURL, "my_ucc", "my_sid", callbacks)
	client.SetTLSConfig(proxyServer.Client().Transport.(*http.Transport).TLSClientConfig)
	client.SetProxyURL(proxyURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go client.Serve(ctx)

	waitCh := make(chan struct{})
	go func() {
		tickReceived.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for tick through HTTPS proxy")
	}

	if capturedTick.InstrumentToken != 11536 {
		t.Errorf("InstrumentToken = %d, want 11536", capturedTick.InstrumentToken)
	}
	if proxyConnectCount.Load() == 0 {
		t.Errorf("proxyConnectCount = 0, expected traffic through HTTPS proxy")
	}
}

// Suppress unused math import if needed
var _ = math.Exp2
