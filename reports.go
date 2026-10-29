package gokotakneo

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/DragoHex/gokotakneo/models"
)

// GetOrders retrieves all open, completed, and rejected orders for the trading session.
// Endpoint: GET <Base URL>/quick/user/orders
func (c *Client) GetOrders(ctx context.Context) ([]models.Order, error) {
	var resp struct {
		Stat string         `json:"stat"`
		Data []models.Order `json:"data"`
	}

	err := c.transport.executeRaw(ctx, http.MethodGet, URIGetOrders, nil, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return []models.Order{}, nil
	}

	return resp.Data, nil
}

// GetOrderBook retrieves all orders for the current trading day (alias for GetOrders).
func (c *Client) GetOrderBook(ctx context.Context) ([]models.Order, error) {
	return c.GetOrders(ctx)
}

// GetOrderByID retrieves order details for a specific order number.
// Endpoint: GET <Base URL>/quick/user/orders/<order_no>
func (c *Client) GetOrderByID(ctx context.Context, orderNo string) (*models.Order, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, fmt.Errorf("%w: order number is required", ErrInvalidInput)
	}

	endpoint := fmt.Sprintf(URIGetOrderByID, orderNo)
	var resp struct {
		Stat string       `json:"stat"`
		Data models.Order `json:"data"`
	}

	err := c.transport.executeRaw(ctx, http.MethodGet, endpoint, nil, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// GetOrderHistory retrieves all state transitions/audit updates for an order.
// Endpoint: POST <Base URL>/quick/user/order/history
func (c *Client) GetOrderHistory(ctx context.Context, orderNo string) ([]models.Order, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, fmt.Errorf("%w: order number is required", ErrInvalidInput)
	}

	payload := map[string]string{
		"nOrdNo": orderNo,
	}

	var resp struct {
		Stat string         `json:"stat"`
		Data []models.Order `json:"data"`
	}

	err := c.transport.ExecuteJData(ctx, http.MethodPost, URIGetOrderHistory, payload, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return []models.Order{}, nil
	}

	return resp.Data, nil
}

// GetTrades retrieves executed fills for the current trading day.
// Endpoint: GET <Base URL>/quick/user/trades
func (c *Client) GetTrades(ctx context.Context) ([]models.Trade, error) {
	var resp struct {
		Stat string         `json:"stat"`
		Data []models.Trade `json:"data"`
	}

	err := c.transport.executeRaw(ctx, http.MethodGet, URIGetTrades, nil, nil, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return []models.Trade{}, nil
	}

	return resp.Data, nil
}
