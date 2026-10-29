package orderstream

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Client manages live order lifecycle events and position updates.
type Client struct {
	rtURL            string
	tradeToken       string
	tradeSid         string
	callbacks        StreamCallbacks
	conn             *websocket.Conn
	lastActivity     atomic.Value
	autoReconnect    bool
	reconnectAttempt int
	cancel           context.CancelFunc
	proxyURL         *url.URL
	mu               sync.Mutex
}

// SetProxyURL sets proxy URL on the order stream client.
func (c *Client) SetProxyURL(rawURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	trimmed := strings.TrimSpace(rawURL)
	if trimmed != "" {
		if u, err := url.Parse(trimmed); err == nil && u.Host != "" {
			c.proxyURL = u
		}
	}
}

// New creates a new order/position streaming client.
func New(rtURL, tradeToken, tradeSid string, callbacks StreamCallbacks) *Client {
	wsURL := rtURL
	if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	} else if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	}

	c := &Client{
		rtURL:         wsURL,
		tradeToken:    tradeToken,
		tradeSid:      tradeSid,
		callbacks:     callbacks,
		autoReconnect: true,
	}
	c.lastActivity.Store(time.Now())
	return c
}

// Close gracefully terminates the order stream connection and stops auto-reconnect.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.autoReconnect = false
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Serve starts the connection and message loop until context cancellation or Close().
func (c *Client) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := c.connectAndRead(ctx)
			if err != nil && c.callbacks.OnError != nil {
				c.callbacks.OnError(err)
			}

			c.mu.Lock()
			shouldReconnect := c.autoReconnect
			c.mu.Unlock()

			if !shouldReconnect {
				return err
			}

			c.reconnectAttempt++
			delay := c.calculateBackoff()
			if c.callbacks.OnReconnect != nil {
				c.callbacks.OnReconnect(c.reconnectAttempt, delay)
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func (c *Client) calculateBackoff() time.Duration {
	base := 500 * time.Millisecond
	delay := base * time.Duration(math.Exp2(float64(c.reconnectAttempt)))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	jitter := time.Duration(rand.Intn(500)) * time.Millisecond
	return delay + jitter
}

func (c *Client) connectAndRead(ctx context.Context) error {
	u, err := url.Parse(c.rtURL)
	if err != nil {
		return fmt.Errorf("invalid rt url: %w", err)
	}

	c.mu.Lock()
	proxyURL := c.proxyURL
	c.mu.Unlock()

	dialer := websocket.DefaultDialer
	if proxyURL != nil {
		dialer = &websocket.Dialer{
			Proxy:            http.ProxyURL(proxyURL),
			HandshakeTimeout: 10 * time.Second,
		}
	}
	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("order stream websocket dial: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.mu.Unlock()
		conn.Close()
	}()

	c.reconnectAttempt = 0
	c.lastActivity.Store(time.Now())

	// Step 1: Send Raw Non-JSON Handshake String (per Kotak spec)
	handshake := fmt.Sprintf("{type:cn,Authorization:%s,Sid:%s,src:WEB}", c.tradeToken, c.tradeSid)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(handshake)); err != nil {
		return fmt.Errorf("send raw handshake: %w", err)
	}

	// Register Ping Handler to update activity timer and respond with Pong
	conn.SetPingHandler(func(appData string) error {
		c.lastActivity.Store(time.Now())
		c.mu.Lock()
		defer c.mu.Unlock()
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	if c.callbacks.OnConnect != nil {
		c.callbacks.OnConnect()
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	readCtx, readCancel := context.WithCancel(ctx)
	defer readCancel()

	// Inactivity Watchdog: 90 seconds threshold
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-readCtx.Done():
				return
			case <-ticker.C:
				last := c.lastActivity.Load().(time.Time)
				if time.Since(last) > 90*time.Second {
					conn.Close()
					return
				}
			}
		}
	}()

	// Ping keep-alive: send Ping every 30 seconds
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-readCtx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				err := conn.WriteControl(websocket.PingMessage, []byte("keepalive"), time.Now().Add(time.Second))
				c.mu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		c.lastActivity.Store(time.Now())

		var envelope struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(data, &envelope); err == nil {
			if envelope.Type == "order" && c.callbacks.OnOrderUpdate != nil {
				var order OrderUpdate
				if err := json.Unmarshal(envelope.Data, &order); err == nil {
					c.callbacks.OnOrderUpdate(order)
				}
			} else if envelope.Type == "position" && c.callbacks.OnPositionUpdate != nil {
				var pos PositionUpdate
				if err := json.Unmarshal(envelope.Data, &pos); err == nil {
					c.callbacks.OnPositionUpdate(pos)
				}
			}
		}
	}
}
