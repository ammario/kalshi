package kalshi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	Resting  OrderStatus = "resting"
	Canceled OrderStatus = "canceled"
	Executed OrderStatus = "executed"
	Pending  OrderStatus = "pending"
)

type OrderAction string

const (
	Buy  OrderAction = "buy"
	Sell OrderAction = "sell"
)

type OrderType string

const (
	MarketOrder OrderType = "market"
	LimitOrder  OrderType = "limit"
)

// CreateOrderRequest is described here:
// https://trading-api.readme.io/reference/createorder.
type CreateOrderRequest struct {
	Action        OrderAction `json:"action,omitempty"`
	BuyMaxCost    Cents       `json:"buy_max_cost,omitempty"`
	Count         int         `json:"count,omitempty"`
	Expiration    *Timestamp  `json:"expiration_ts,omitempty"`
	NoPrice       Cents       `json:"no_price,omitempty"`
	YesPrice      Cents       `json:"yes_price,omitempty"`
	Ticker        string      `json:"ticker,omitempty"`
	ClientOrderID string      `json:"client_order_id,omitempty"`
	Type          OrderType   `json:"type"`
	Side          Side        `json:"side"`
}

// SetPrice sets the price of the order based on its side.
func (c *CreateOrderRequest) SetPrice(p Cents) {
	switch c.Side {
	case Yes:
		c.YesPrice = p
	case No:
		c.NoPrice = p
	default:
		panic("invalid side: " + string(c.Side))
	}
}

func (o *CreateOrderRequest) Price() Cents {
	switch o.Side {
	case Yes:
		return o.YesPrice
	case No:
		return o.NoPrice
	default:
		panic("invalid side: " + string(o.Side))
	}
}

// String returns a human-readable representation of the order.
func (c *CreateOrderRequest) String() string {
	var price Cents
	if c.Side == Yes {
		price = c.YesPrice
	} else {
		price = c.NoPrice
	}
	expire := "never"
	if c.Expiration != nil {
		expire = fmt.Sprintf("in %v", time.Until(c.Expiration.Time()))
	}
	return fmt.Sprintf(
		"%v %v %v at %v, expires %s, max cost is %v",
		c.Action, c.Count, strings.ToUpper(string(c.Side)), price, expire, c.BuyMaxCost,
	)
}

// When passed to `CreateOrder`, the order will attempt to partially or completely fill
// and the remaining unfilled quantity will be cancelled.
// This is also known as Immediate-or-Cancel (IOC).
func OrderExecuteImmediateOrCancel() *Timestamp {
	t := Timestamp(time.Now().AddDate(-10, 0, 0))
	return &t
}

// ExpireAfter is a helper function for creating an expiration timestamp
// some duration after the current time.
func ExpireAfter(duration time.Duration) *Timestamp {
	t := Timestamp(time.Now().Add(duration))
	return &t
}

// When passed to `CreateOrder`, the order won't expire until explicitly cancelled.
// This is also known as Good 'Till Cancelled (GTC). This function just returns `nil`.
func OrderGoodTillCanceled() *Timestamp {
	return nil
}

// CreateOrder is described here:
// https://trading-api.readme.io/reference/createorder.
func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
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
	Ticker string      `url:"ticker,omitempty"`
	Status OrderStatus `url:"status,omitempty"`
}

// Order is described here:
// https://trading-api.readme.io/reference/getorders.
type Order struct {
	Action           OrderAction `json:"action"`
	ClientOrderID    string      `json:"client_order_id"`
	CloseCancelCount int         `json:"close_cancel_count"`
	CreatedTime      *Time       `json:"created_time"`
	DecreaseCount    int         `json:"decrease_count"`
	ExpirationTime   *Time       `json:"expiration_time"`
	FccCancelCount   int         `json:"fcc_cancel_count"`
	LastUpdateTime   *Time       `json:"last_update_time"`
	MakerFillCount   int         `json:"maker_fill_count"`
	NoPrice          Cents       `json:"no_price"`
	OrderID          string      `json:"order_id"`
	PlaceCount       int         `json:"place_count"`
	QueuePosition    int         `json:"queue_position"`
	RemainingCount   int         `json:"remaining_count"`
	Side             Side        `json:"side"`
	Status           OrderStatus `json:"status"`
	TakerFees        Cents       `json:"taker_fees"`
	TakerFillCost    Cents       `json:"taker_fill_cost"`
	TakerFillCount   int         `json:"taker_fill_count"`
	Ticker           string      `json:"ticker"`
	Type             OrderType   `json:"type"`
	UserID           string      `json:"user_id"`
	YesPrice         Cents       `json:"yes_price"`
}

func (o *Order) Price() Cents {
	if o.Side == Yes {
		return o.YesPrice
	} else if o.Side == No {
		return o.NoPrice
	}
	panic("invalid side: " + string(o.Side))
}

// Orders is described here:
// https://trading-api.readme.io/reference/getorders.
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
	Action      OrderAction `json:"action"`
	Count       int         `json:"count"`
	CreatedTime time.Time   `json:"created_time"`
	IsTaker     bool        `json:"is_taker"`
	NoPrice     Cents       `json:"no_price"`
	OrderID     string      `json:"order_id"`
	Side        Side        `json:"side"`
	Ticker      string      `json:"ticker"`
	TradeID     string      `json:"trade_id"`
	YesPrice    Cents       `json:"yes_price"`
}

// FillsRequest is described here:
// https://trading-api.readme.io/reference/getfills.
type FillsRequest struct {
	CursorRequest
	Ticker  string    `url:"ticker,omitempty"`
	OrderID string    `url:"order_id,omitempty"`
	MinTS   Timestamp `url:"min_ts,omitempty"`
	MaxTS   Timestamp `url:"max_ts,omitempty"`
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

type SettlementStatus string

const (
	StatusAll       SettlementStatus = "all"
	StatusSettled   SettlementStatus = "settled"
	StatusUnsettled SettlementStatus = "unsettled"
)

// Position is described here:
// https://trading-api.readme.io/reference/getpositions.
type PositionsRequest struct {
	CursorRequest
	Limit            int              `url:"limit,omitempty"`
	SettlementStatus SettlementStatus `url:"settlement_status,omitempty"`
	Ticker           string           `url:"ticker,omitempty"`
	EventTicker      string           `url:"event_ticker,omitempty"`
}

// EventPosition is described here:
// https://trading-api.readme.io/reference/getpositions.
type EventPosition struct {
	EventExposure     Cents  `json:"event_exposure"`
	EventTicker       string `json:"event_ticker"`
	FeesPaid          Cents  `json:"fees_paid"`
	RealizedPnl       Cents  `json:"realized_pnl"`
	RestingOrderCount int    `json:"resting_order_count"`
	TotalCost         Cents  `json:"total_cost"`
}

// MarketPosition is described here:
// https://trading-api.readme.io/reference/getpositions.
type MarketPosition struct {
	// Fees paid on fill orders, in cents.
	FeesPaid Cents `json:"fees_paid"`
	// Number of contracts bought in this market. Negative means NO contracts and positive means YES contracts.
	Position int `json:"position"`
	// Locked in profit and loss, in cents.
	RealizedPnl Cents `json:"realized_pnl"`
	// Aggregate size of resting orders in contract units.
	RestingOrdersCount int `json:"resting_orders_count"`
	// Unique identifier for the market.
	Ticker string `json:"ticker"`
	// Total spent on this market in cents.
	TotalTraded Cents `json:"total_traded"`
	// Cost of the aggregate market position in cents.
	MarketExposure Cents `json:"market_exposure"`
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
