package gokotakneo

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/DragoHex/gokotakneo/models"
)

// GetQuotes retrieves live quote snapshots for specified symbols with an optional filter.
// Endpoint: GET <Base URL>/script-details/1.0/quotes/neosymbol/<query>[/<filter>]
// Valid filters: "all", "ltp", "ohlc", "depth", "circuit_limits", "52W", "oi", "scrip_details"
func (c *Client) GetQuotes(ctx context.Context, queries []string, filter string) (map[string]models.Quote, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("%w: at least one symbol query required", ErrInvalidInput)
	}

	queryString := strings.Join(queries, ",")
	endpoint := fmt.Sprintf(URIQuotes, queryString)
	if filter != "" {
		endpoint = fmt.Sprintf("%s/%s", endpoint, filter)
	}

	headers := http.Header{}
	headers.Set("Authorization", c.accessToken)

	var resp struct {
		Stat string         `json:"stat"`
		Data []models.Quote `json:"data"`
	}

	err := c.transport.executeRaw(ctx, http.MethodGet, endpoint, nil, headers, &resp)
	if err != nil {
		return nil, err
	}

	result := make(map[string]models.Quote, len(resp.Data))
	for _, q := range resp.Data {
		result[q.TradingSymbol] = q
	}

	return result, nil
}

// GetLTP retrieves lightweight last traded prices for specified symbols.
func (c *Client) GetLTP(ctx context.Context, queries []string) (map[string]models.LTPQuote, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("%w: at least one symbol query required", ErrInvalidInput)
	}

	queryString := strings.Join(queries, ",")
	endpoint := fmt.Sprintf(URILTP, queryString)

	headers := http.Header{}
	headers.Set("Authorization", c.accessToken)

	var resp struct {
		Stat string            `json:"stat"`
		Data []models.LTPQuote `json:"data"`
	}

	err := c.transport.executeRaw(ctx, http.MethodGet, endpoint, nil, headers, &resp)
	if err != nil {
		return nil, err
	}

	result := make(map[string]models.LTPQuote, len(resp.Data))
	for _, ltp := range resp.Data {
		result[ltp.TradingSymbol] = ltp
	}

	return result, nil
}
