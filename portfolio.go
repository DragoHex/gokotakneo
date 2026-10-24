package gokotakneo

import (
	"context"
	"net/http"

	"github.com/DragoHex/gokotakneo/models"
)

// GetPositions retrieves consolidated open and closed positions for the trading day.
// Endpoint: GET <Base URL>/quick/user/positions
func (c *Client) GetPositions(ctx context.Context) ([]models.Position, error) {
	var resp struct {
		Stat string            `json:"stat"`
		Data []models.Position `json:"data"`
	}

	err := c.transport.executeRaw(ctx, http.MethodGet, URIGetPositions, nil, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return []models.Position{}, nil
	}

	return resp.Data, nil
}

// GetHoldings retrieves long-term Demat custody portfolio holdings.
// Endpoint: GET <Base URL>/portfolio/v1/holdings
func (c *Client) GetHoldings(ctx context.Context) ([]models.Holding, error) {
	var resp struct {
		Data []models.Holding `json:"data"`
	}

	err := c.transport.executeRaw(ctx, http.MethodGet, URIGetHoldings, nil, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return []models.Holding{}, nil
	}

	return resp.Data, nil
}

// GetLimits retrieves real-time available limits, margins, collateral, and cash balances.
// Endpoint: POST <Base URL>/quick/user/limits
func (c *Client) GetLimits(ctx context.Context, seg, exch, prod string) (*models.Limits, error) {
	if seg == "" {
		seg = "ALL"
	}
	if exch == "" {
		exch = "ALL"
	}
	if prod == "" {
		prod = "ALL"
	}

	payload := map[string]string{
		"seg":  seg,
		"exch": exch,
		"prod": prod,
	}

	var limits models.Limits
	err := c.transport.ExecuteJData(ctx, http.MethodPost, URIGetLimits, payload, &limits)
	if err != nil {
		return nil, err
	}

	return &limits, nil
}
