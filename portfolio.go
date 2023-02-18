package kalshi

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		"%v %v %v at %v cents, expires in %v, max cost is %v",
		c.Action, c.Count, strings.ToUpper(string(c.Side)), price, time.Until(c.Expiration.Time()), c.BuyMaxCost,
	)
}

// CreateOrder is described here:
// https://trading-api.readme.io/reference/createorder.
func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	if req.Expiration.Time().IsZero() {
		// Otherwise, API will fail with obscure error.
		return nil, fmt.Errorf("expiration is required")
	}

	if req.ClientOrderID == "" {
		req.ClientOrderID = uuid.New().String()
	}

	var resp struct {
		Order Order `json:"order"`
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

// Order is described here:
// https://trading-api.readme.io/reference/getorders.
type Order struct {
	Action           string `json:"action"`
	ClientOrderID    string `json:"client_order_id"`
	CloseCancelCount int    `json:"close_cancel_count"`
	CreatedTime      *Time  `json:"created_time"`
	DecreaseCount    int    `json:"decrease_count"`
	ExpirationTime   *Time  `json:"expiration_time"`
	FccCancelCount   int    `json:"fcc_cancel_count"`
	LastUpdateTime   *Time  `json:"last_update_time"`
	MakerFillCount   int    `json:"maker_fill_count"`
	NoPrice          int    `json:"no_price"`
	OrderID          string `json:"order_id"`
	PlaceCount       int    `json:"place_count"`
	QueuePosition    int    `json:"queue_position"`
	RemainingCount   int    `json:"remaining_count"`
	Side             string `json:"side"`
	Status           string `json:"status"`
	TakerFees        int    `json:"taker_fees"`
	TakerFillCost    int    `json:"taker_fill_cost"`
	TakerFillCount   int    `json:"taker_fill_count"`
	Ticker           string `json:"ticker"`
	Type             string `json:"type"`
	UserID           string `json:"user_id"`
	YesPrice         int    `json:"yes_price"`
}

type OrdersResponse struct {
	CursorResponse
	Orders []Order `json:"orders"`
}

// Orders is described here:
// https://trading-api.readme.io/reference/getorders
func (c *Client) Orders(ctx context.Context, req OrdersRequest) (*OrdersResponse, error) {
	var resp OrdersResponse
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "portfolio/orders",
		QueryParams:  req,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
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

// Fill is described here:
// https://trading-api.readme.io/reference/getfills.
type Fill struct {
	Action      string    `json:"action"`
	Count       int       `json:"count"`
	CreatedTime time.Time `json:"created_time"`
	IsTaker     bool      `json:"is_taker"`
	NoPrice     int       `json:"no_price"`
	OrderID     string    `json:"order_id"`
	Side        string    `json:"side"`
	Ticker      string    `json:"ticker"`
	TradeID     string    `json:"trade_id"`
	YesPrice    int       `json:"yes_price"`
}

// FillsRequest is described here:
// https://trading-api.readme.io/reference/getfills.
type FillsRequest struct {
	CursorRequest
	Ticker  string    `url:"ticker,omitempty"`
	OrderID string    `url:"order_id,omitempty"`
	MinTS   timestamp `url:"min_ts,omitempty"`
	MaxTS   timestamp `url:"max_ts,omitempty"`
}

// FillsResponse is described here:
// https://trading-api.readme.io/reference/getfills.
type FillsResponse struct {
	CursorResponse
	Fills []Fill `json:"fills"`
}

// Fills is described here:
// https://trading-api.readme.io/reference/getfills.
func (c *Client) Fills(ctx context.Context, req FillsRequest) (*FillsResponse, error) {
	var resp FillsResponse
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "portfolio/fills",
		QueryParams:  req,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Order is described here:
// https://trading-api.readme.io/reference/getorder.
func (c *Client) Order(ctx context.Context, orderID string) (*Order, error) {
	var resp struct {
		Order Order `json:"order"`
	}
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "portfolio/orders/" + orderID,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp.Order, nil
}

// CancelOrder is described here:
// https://trading-api.readme.io/reference/cancelorder.
func (c *Client) CancelOrder(ctx context.Context, orderID string) (*Order, error) {
	var resp struct {
		Order Order `json:"order"`
	}
	err := c.request(ctx, request{
		Method:       "DELETE",
		Endpoint:     "portfolio/orders/" + orderID,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp.Order, nil
}

// DecreaseOrder is described here:
// https://trading-api.readme.io/reference/decreaseorder.
type DecreaseOrderRequest struct {
	ReduceBy int `json:"reduce_by,omitempty"`
	ReduceTo int `json:"reduce_to,omitempty"`
}

// DecreaseOrder is described here:
// https://trading-api.readme.io/reference/decreaseorder.
func (c *Client) DecreaseOrder(ctx context.Context, orderID string, req DecreaseOrderRequest) (*Order, error) {
	var resp struct {
		Order Order `json:"order"`
	}
	err := c.request(ctx, request{
		Method:       "POST",
		Endpoint:     "portfolio/orders/" + orderID + "/decrease",
		JSONRequest:  req,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp.Order, nil
}

// Position is described here:
// https://trading-api.readme.io/reference/getpositions.
type PositionsRequest struct {
	CursorRequest
	// SettlementStatus is one of "all", "settled", or "unsettled".
	SettlementStatus string `url:"settlement_status,omitempty"`
	Ticker           string `url:"ticker,omitempty"`
	EventTicker      string `url:"event_ticker,omitempty"`
}

// EventPosition is described here:
// https://trading-api.readme.io/reference/getpositions.
type EventPosition struct {
	EventExposure     int    `json:"event_exposure"`
	EventTicker       string `json:"event_ticker"`
	FeesPaid          int    `json:"fees_paid"`
	RealizedPnl       int    `json:"realized_pnl"`
	RestingOrderCount int    `json:"resting_order_count"`
	TotalCost         int    `json:"total_cost"`
}

// MarketPosition is described here:
// https://trading-api.readme.io/reference/getpositions.
type MarketPosition struct {
	FeesPaid           int    `json:"fees_paid"`
	FinalPosition      int    `json:"final_position"`
	FinalPositionCost  int    `json:"final_position_cost"`
	IsSettled          bool   `json:"is_settled"`
	LastPosition       int    `json:"last_position"`
	MarketID           string `json:"market_id"`
	Position           int    `json:"position"`
	PositionCost       int    `json:"position_cost"`
	RealizedPnl        int    `json:"realized_pnl"`
	RestingOrdersCount int    `json:"resting_orders_count"`
	TotalCost          int    `json:"total_cost"`
	UserID             string `json:"user_id"`
	Volume             int    `json:"volume"`
}

// PositionsResponse is described here:
// https://trading-api.readme.io/reference/getpositions.
type PositionsResponse struct {
	CursorResponse
	EventPositions  []EventPosition  `json:"event_positions"`
	MarketPositions []MarketPosition `json:"market_positions"`
}

// Positions is described here:
// https://trading-api.readme.io/reference/getpositions.
func (c *Client) Positions(ctx context.Context, req PositionsRequest) (*PositionsResponse, error) {
	var resp PositionsResponse
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "portfolio/positions",
		QueryParams:  req,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PortfolioSettlements is described here:
// https://trading-api.readme.io/reference/getportfoliosettlements.
type SettlementsRequest struct {
	CursorRequest
}

// Settlement is described here:
// https://trading-api.readme.io/reference/getportfoliosettlements.
type Settlement struct {
	MarketResult string    `json:"market_result"`
	NoCount      int       `json:"no_count"`
	NoTotalCost  int       `json:"no_total_cost"`
	Revenue      int       `json:"revenue"`
	SettledTime  time.Time `json:"settled_time"`
	Ticker       string    `json:"ticker"`
	YesCount     int       `json:"yes_count"`
	YesTotalCost int       `json:"yes_total_cost"`
}

// SettlementsResponse is described here:
// https://trading-api.readme.io/reference/getportfoliosettlements.
type SettlementsResponse struct {
	CursorResponse
	Settlements []Settlement `json:"settlements"`
}

// Settlements is described here:
// https://trading-api.readme.io/reference/getportfoliosettlements.
func (c *Client) Settlements(ctx context.Context, req SettlementsRequest) (*SettlementsResponse, error) {
	var resp SettlementsResponse
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "portfolio/settlements",
		QueryParams:  req,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
