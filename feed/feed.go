package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Client manages live quotes WebSocket connection to Kotak-Neo feedUrl.
type Client struct {
	feedURL          string
	ucc              string
	sessionSid       string
	callbacks        FeedCallbacks
	conn             *websocket.Conn
	parser           *Parser
	subscriptions    map[string]bool
	subMu            sync.RWMutex
	lastActivity     atomic.Value
	autoReconnect    bool
	reconnectAttempt int
	cancel           context.CancelFunc
}

// NewClient creates a market quotes streaming client.
func NewClient(feedURL, ucc, sessionSid string, callbacks FeedCallbacks) *Client {
	wsURL := feedURL
	if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	} else if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	}

	c := &Client{
		feedURL:       wsURL,
		ucc:           ucc,
		sessionSid:    sessionSid,
		callbacks:     callbacks,
		parser:        NewParser(),
		subscriptions: make(map[string]bool),
		autoReconnect: true,
	}
	c.lastActivity.Store(time.Now())
	return c
}

// Serve connects to feed and processes incoming frames until context cancellation.
func (c *Client) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
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

			if !c.autoReconnect {
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
	u, err := url.Parse(c.feedURL)
	if err != nil {
		return fmt.Errorf("invalid feed url: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("feed websocket dial: %w", err)
	}
	c.conn = conn
	defer conn.Close()

	c.reconnectAttempt = 0
	c.lastActivity.Store(time.Now())

	// Step 1: Send JSON Auth Handshake
	authPayload := map[string]interface{}{
		"user":        c.ucc,
		"auth":        c.sessionSid,
		"format":      "native_batch",
		"source":      "NEOTRADEAPI",
		"sdk_version": 1.1,
	}
	if err := conn.WriteJSON(authPayload); err != nil {
		return fmt.Errorf("send feed auth handshake: %w", err)
	}

	if c.callbacks.OnConnect != nil {
		c.callbacks.OnConnect()
	}

	// Resubscribe cached tokens
	c.resubscribe()

	var wg sync.WaitGroup
	defer wg.Wait()

	readCtx, readCancel := context.WithCancel(ctx)
	defer readCancel()

	// Watchdog Goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-readCtx.Done():
				return
			case <-ticker.C:
				last := c.lastActivity.Load().(time.Time)
				if time.Since(last) > 6*time.Second {
					c.conn.Close()
					return
				}
			}
		}
	}()

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			return err
		}
		c.lastActivity.Store(time.Now())

		if msgType == websocket.BinaryMessage {
			ticks, err := c.parser.ParseBatch(data, "nse_cm")
			if err == nil {
				for _, t := range ticks {
					if c.callbacks.OnTick != nil {
						c.callbacks.OnTick(t)
					}
				}
			}
		} else if msgType == websocket.TextMessage {
			// Parse dividers from JSON ack
			var ack struct {
				Dividers map[string]float64 `json:"dividers"`
			}
			if err := json.Unmarshal(data, &ack); err == nil && len(ack.Dividers) > 0 {
				for seg, div := range ack.Dividers {
					c.parser.SetDivider(seg, div)
				}
			}
		}
	}
}

func (c *Client) resubscribe() {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	if len(c.subscriptions) == 0 || c.conn == nil {
		return
	}
	var symbols []string
	for s := range c.subscriptions {
		symbols = append(symbols, s)
	}
	c.conn.WriteJSON(map[string]interface{}{
		"type":    "subscribe",
		"symbols": symbols,
	})
}

// Subscribe adds symbols to the live feed.
func (c *Client) Subscribe(symbols []string) error {
	c.subMu.Lock()
	for _, s := range symbols {
		c.subscriptions[s] = true
	}
	c.subMu.Unlock()

	if c.conn != nil {
		return c.conn.WriteJSON(map[string]interface{}{
			"type":    "subscribe",
			"symbols": symbols,
		})
	}
	return nil
}
