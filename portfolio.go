package kalshi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/google/uuid"
)

// CreateOrderRequest is described here:
// https://trading-api.readme.io/reference/createorder.
type CreateOrderRequest struct {
	// Action is either "buy" or "sell"
	Action        string    `json:"action,omitempty"`
	BuyMaxCost    int       `json:"buy_max_cost,omitempty"`
	Count         int       `json:"count,omitempty"`
	Expiration    timestamp `json:"expiration_ts,omitempty"`
	NoPrice       int       `json:"no_price,omitempty"`
	YesPrice      int       `json:"yes_price,omitempty"`
	Ticker        string    `json:"ticker,omitempty"`
	ClientOrderID string    `json:"client_order_id,omitempty"`
	// Type is either "market" or "limit"
	Type string `json:"type"`
	Side Side   `json:"side"`
}

func (c *CreateOrderRequest) String() string {
	var price int
	if c.Side == Yes {
		price = c.YesPrice
	} else {
		price = c.NoPrice
	}
	return fmt.Sprintf(
		"BUY %v %v at %v cents, expires in %v, max cost is %v",
		c.Count, strings.ToUpper(string(c.Side)), price, time.Until(c.Expiration.Time()), c.BuyMaxCost,
	)
}

// CreateOrder is described here:
// https://trading-api.readme.io/reference/createorder.
func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest) (*UserOrder, error) {
	if req.Expiration.Time().IsZero() {
		// Otherwise, API will fail with obscure error.
		return nil, fmt.Errorf("expiration is required")
	}

	if req.ClientOrderID == "" {
		req.ClientOrderID = uuid.New().String()
	}

	var resp struct {
		Order UserOrder `json:"order"`
	}
	err := c.request(ctx, request{
		Method:       "POST",
		Endpoint:     "portfolio/orders",
		JSONRequest:  req,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}

	return &resp.Order, nil
}

// OrdersRequest is described here:
// https://trading-api.readme.io/reference/getorders
type OrdersRequest struct {
	Ticker string `url:"ticker,omitempty"`
	Status string `url:"status,omitempty"`
}

type UserOrder struct {
	Action     string  `json:"action"`
	OrderType  string  `json:"order_type,omitempty"`
	OrderID    string  `json:"client_order_id,omitempty"`
	Price      float64 `json:"price,omitempty"`
	Status     string  `json:"status,omitempty"`
	Ticker     string  `json:"ticker"`
	Expiration *Time   `json:"expiration_time,omitempty"`
}

// Orders is described here:
// https://trading-api.readme.io/reference/getorders
func (c *Client) Orders(ctx context.Context, req OrdersRequest) ([]UserOrder, error) {
	var resp struct {
		Orders []UserOrder `json:"orders"`
	}
	v, err := query.Values(req)
	if err != nil {
		return nil, err
	}
	err = c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "portfolio/orders/?" + v.Encode(),
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// Balance is described here:
// https://trading-api.readme.io/reference/getbalance.
func (c *Client) Balance(ctx context.Context) (Cents, error) {
	var resp struct {
		Balance Cents `json:"balance"`
	}
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "portfolio/balance",
		JSONResponse: &resp,
	})
	if err != nil {
		return -1, err
	}
	return resp.Balance, nil
}
