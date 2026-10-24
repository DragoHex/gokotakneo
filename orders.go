package gokotakneo

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/DragoHex/gokotakneo/models"
)

// PlaceOrder executes a new regular or AMO order across supported segments and products.
// Endpoint: POST <Base URL>/quick/order/rule/ms/place
func (c *Client) PlaceOrder(ctx context.Context, params models.OrderParams) (*models.OrderResponse, error) {
	if params.Ts == "" {
		return nil, fmt.Errorf("%w: trading symbol (ts) is required", ErrInvalidInput)
	}

	var resp models.OrderResponse
	err := c.transport.ExecuteJData(ctx, http.MethodPost, URIPlaceOrder, params, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// ModifyOrder modifies an open or pending order's price, trigger price, or quantity.
// Endpoint: POST <Base URL>/quick/order/vr/modify
func (c *Client) ModifyOrder(ctx context.Context, params models.OrderModifyParams) (*models.OrderResponse, error) {
	if strings.TrimSpace(params.OrderNo) == "" {
		return nil, fmt.Errorf("%w: order number (no) is required", ErrInvalidInput)
	}

	var resp models.OrderResponse
	err := c.transport.ExecuteJData(ctx, http.MethodPost, URIModifyOrder, params, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// CancelOrder revokes an open order.
// Endpoint: POST <Base URL>/quick/order/cancel
func (c *Client) CancelOrder(ctx context.Context, orderNo string, isAMO bool, tradingSymbol string) (*models.OrderResponse, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, fmt.Errorf("%w: order number (on) is required", ErrInvalidInput)
	}

	amoFlag := "NO"
	if isAMO {
		amoFlag = "YES"
	}

	payload := map[string]string{
		"on": orderNo,
		"am": amoFlag,
		"ts": tradingSymbol,
	}

	var resp models.OrderResponse
	err := c.transport.ExecuteJData(ctx, http.MethodPost, URICancelOrder, payload, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
