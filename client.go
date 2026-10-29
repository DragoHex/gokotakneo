package gokotakneo

import (
	"strings"
	"sync"

	"github.com/shopspring/decimal"
)

// Client represents the Kotak-Neo Trade API client.
type Client struct {
	transport     *Transport
	accessToken   string
	baseURL       string
	feedURL       string
	rtURL         string
	authenticated bool
	mu            sync.RWMutex
}

// Option allows customizing Client configuration.
type Option func(*Client)

// WithBaseURL overrides the default base URL.
func WithBaseURL(uri string) Option {
	return func(c *Client) {
		c.baseURL = uri
		c.transport.SetBaseURI(uri)
	}
}

// WithHTTPClient injects a custom HTTP client into the transport seam.
func WithHTTPClient(httpClient HTTPClient) Option {
	return func(c *Client) {
		c.transport.client = httpClient
	}
}

// New creates a new Kotak-Neo client with portal access token.
func New(accessToken string, opts ...Option) *Client {
	c := &Client{
		accessToken: accessToken,
		baseURL:     "https://mis.kotaksecurities.com",
		transport:   NewDefaultTransport("https://mis.kotaksecurities.com"),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// SetGateways dynamically updates the gateway endpoints returned after Step 2 authentication.
func (c *Client) SetGateways(baseURL, feedURL, rtURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = baseURL
	c.feedURL = feedURL
	c.rtURL = rtURL
	c.transport.SetBaseURI(baseURL)
}

// SetSessionTokens updates active trade session credentials and authenticated state.
func (c *Client) SetSessionTokens(tradeToken, tradeSid string) {
	c.transport.SetSessionHeaders(tradeToken, tradeSid)
	c.mu.Lock()
	c.authenticated = strings.TrimSpace(tradeToken) != "" && strings.TrimSpace(tradeSid) != ""
	c.mu.Unlock()
}

// BaseURL returns the currently configured base URL.
func (c *Client) BaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// FeedURL returns the market quotes WebSocket endpoint.
func (c *Client) FeedURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.feedURL
}

// RTURL returns the order/position real-time WebSocket endpoint.
func (c *Client) RTURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rtURL
}

// MustDecimal parses string to decimal.Decimal, panics on invalid input.
func MustDecimal(val string) decimal.Decimal {
	return decimal.RequireFromString(val)
}
