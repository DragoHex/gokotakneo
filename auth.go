package gokotakneo

import (
	"context"
	"net/http"
	"strings"

	"github.com/DragoHex/gokotakneo/models"
)

// LoginTOTP executes Step 1 authentication by verifying mobile, UCC, and TOTP.
// Endpoint: POST https://mis.kotaksecurities.com/login/1.0/tradeApiLogin
func (c *Client) LoginTOTP(ctx context.Context, mobileNumber, ucc, totp string) (*models.ViewSession, error) {
	mobileNumber = strings.TrimSpace(mobileNumber)
	ucc = strings.TrimSpace(ucc)
	totp = strings.TrimSpace(totp)

	payload := map[string]string{
		"mobileNumber": mobileNumber,
		"ucc":          ucc,
		"totp":         totp,
	}

	headers := http.Header{}
	headers.Set("Authorization", c.accessToken)
	headers.Set("neo-fin-key", "neotradeapi")

	var resp struct {
		Data models.ViewSession `json:"data"`
	}

	err := c.transport.ExecuteJSON(ctx, http.MethodPost, URILoginTOTP, payload, headers, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// ValidateMPIN executes Step 2 authentication using 6-digit MPIN and View credentials.
// Endpoint: POST https://mis.kotaksecurities.com/login/1.0/tradeApiValidate
func (c *Client) ValidateMPIN(ctx context.Context, viewToken, viewSid, mpin string) (*models.TradeSession, error) {
	mpin = strings.TrimSpace(mpin)

	payload := map[string]string{
		"mpin": mpin,
	}

	headers := http.Header{}
	headers.Set("Authorization", c.accessToken)
	headers.Set("neo-fin-key", "neotradeapi")
	headers.Set("Auth", viewToken)
	headers.Set("sid", viewSid)

	var resp struct {
		Data models.TradeSession `json:"data"`
	}

	err := c.transport.ExecuteJSON(ctx, http.MethodPost, URIValidateMPIN, payload, headers, &resp)
	if err != nil {
		return nil, err
	}

	// Update client with dynamic gateways and trade credentials
	c.SetGateways(resp.Data.BaseURL, resp.Data.FeedURL, resp.Data.RTURL)
	c.SetSessionTokens(resp.Data.Token, resp.Data.Sid)

	c.mu.Lock()
	c.authenticated = true
	c.mu.Unlock()

	return &resp.Data, nil
}

// IsAuthenticated reports whether active trade credentials have been established.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authenticated
}

// Logout clears active session credentials and marks client unauthenticated.
func (c *Client) Logout(ctx context.Context) error {
	c.mu.Lock()
	c.authenticated = false
	c.mu.Unlock()
	c.SetSessionTokens("", "")
	return nil
}
