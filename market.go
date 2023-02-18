package kalshi

import (
	"context"
	"fmt"
	"time"
)

type Event struct {
	Category          string    `json:"category"`
	EventTicker       string    `json:"event_ticker"`
	MutuallyExclusive bool      `json:"mutually_exclusive"`
	SeriesTicker      string    `json:"series_ticker"`
	StrikeDate        time.Time `json:"strike_date"`
	StrikePeriod      string    `json:"strike_period"`
	SubTitle          string    `json:"sub_title"`
	Title             string    `json:"title"`
}

// EventsResponse is described here:
// https://trading-api.readme.io/reference/getevents.
type EventsResponse struct {
	CursorResponse
	Events []Event `json:"events"`
}

// EventsRequest is described here:
// https://trading-api.readme.io/reference/getevents.
type EventsRequest struct {
	CursorRequest
	// Status is one of "open", "closed", or "settled".
	Status       string `url:"status,omitempty"`
	SeriesTicker string `url:"series_ticker,omitempty"`
}

// Events is described here:
// https://trading-api.readme.io/reference/getevents.
func (c *Client) Events(ctx context.Context, req EventsRequest) (*EventsResponse, error) {
	var resp EventsResponse

	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "events",
		QueryParams:  req,
		JSONRequest:  nil,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// EventResponse is described here:
// https://trading-api.readme.io/reference/getevent.
type EventResponse struct {
	Event   Event    `json:"event"`
	Markets []Market `json:"markets"`
}

// GetEvent is described here:
// https://trading-api.readme.io/reference/getevent.
func (c *Client) GetEvent(ctx context.Context, event string) (*EventResponse, error) {
	var resp EventResponse

	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "events/" + event,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetMarketsRequest is described here:
// https://trading-api.readme.io/reference/getmarkets.
type GetMarketsRequest struct {
	CursorRequest
	EventTicker  string `url:"event_ticker,omitempty"`
	SeriesTicker string `url:"series_ticker,omitempty"`
	MaxCloseTs   int    `url:"max_close_ts,omitempty"`
	MinCloseTs   int    `url:"min_close_ts,omitempty"`
	// Status is one of "open", "closed", and "settled"
	Status  string   `url:"status,omitempty"`
	Tickers []string `url:"status,omitempty"`
}

type Market struct {
	Ticker          string    `json:"ticker"`
	EventTicker     string    `json:"event_ticker"`
	Subtitle        string    `json:"subtitle"`
	OpenTime        time.Time `json:"open_time"`
	CloseTime       time.Time `json:"close_time"`
	ExpirationTime  time.Time `json:"expiration_time"`
	Status          string    `json:"status"`
	YesBid          int       `json:"yes_bid"`
	YesAsk          int       `json:"yes_ask"`
	NoBid           int       `json:"no_bid"`
	NoAsk           int       `json:"no_ask"`
	LastPrice       int       `json:"last_price"`
	PreviousYesBid  int       `json:"previous_yes_bid"`
	PreviousYesAsk  int       `json:"previous_yes_ask"`
	PreviousPrice   int       `json:"previous_price"`
	Volume          int       `json:"volume"`
	Volume24H       int       `json:"volume_24h"`
	Liquidity       int       `json:"liquidity"`
	OpenInterest    int       `json:"open_interest"`
	Result          string    `json:"result"`
	CanCloseEarly   bool      `json:"can_close_early"`
	ExpirationValue string    `json:"expiration_value"`
	Category        string    `json:"category"`
	RiskLimitCents  int       `json:"risk_limit_cents"`
	StrikeType      string    `json:"strike_type"`
	FloorStrike     float64   `json:"floor_strike,omitempty"`
	CapStrike       float64   `json:"cap_strike,omitempty"`
}

// MarketsResponse is described here:
// https://trading-api.readme.io/reference/getmarkets.
type MarketsResponse struct {
	Markets []Market `json:"markets,omitempty"`
	CursorResponse
}

// Markets is described here:
// https://trading-api.readme.io/reference/getmarkets.
func (c *Client) Markets(
	ctx context.Context,
	req GetMarketsRequest,
) (*MarketsResponse, error) {
	var resp MarketsResponse

	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "markets",
		QueryParams:  req,
		JSONRequest:  nil,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type Trade struct {
	Count       int       `json:"count"`
	CreatedTime time.Time `json:"created_time"`
	NoPrice     int       `json:"no_price"`
	TakerSide   string    `json:"taker_side"`
	Ticker      string    `json:"ticker"`
	TradeID     string    `json:"trade_id"`
	YesPrice    int       `json:"yes_price"`
}

// TradesResponse is described here:
// https://trading-api.readme.io/reference/gettrades.
type TradesResponse struct {
	CursorResponse
	Trades []Trade `json:"trades,omitempty"`
}

// TradesRequest is described here:
// https://trading-api.readme.io/reference/gettrades.
type TradesRequest struct {
	CursorRequest
	Trades []Trade `json:"trades"`
}

// Trades is described here:
// https://trading-api.readme.io/reference/gettrades.
func (c *Client) Trades(
	ctx context.Context,
	req TradesRequest,
) (*TradesResponse, error) {
	var resp TradesResponse

	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "markets/trades",
		QueryParams:  req,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Market is described here:
// https://trading-api.readme.io/reference/getmarket.
func (c *Client) Market(ctx context.Context, ticker string) (*Market, error) {
	var resp struct {
		Market Market `json:"market"`
	}
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     fmt.Sprintf("markets/%s", ticker),
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp.Market, nil
}

// MarketHistory is described here:
// https://trading-api.readme.io/reference/getmarkethistory.
type MarketHistory struct {
	NoAsk        int       `json:"no_ask"`
	NoBid        int       `json:"no_bid"`
	OpenInterest int       `json:"open_interest"`
	Ts           timestamp `json:"ts"`
	Volume       int       `json:"volume"`
	YesAsk       int       `json:"yes_ask"`
	YesBid       int       `json:"yes_bid"`
	YesPrice     int       `json:"yes_price"`
}

// MarketHistoryResponse is described here:
// https://trading-api.readme.io/reference/getmarkethistory.
type MarketHistoryResponse struct {
	CursorResponse
	History []MarketHistory `json:"history"`
	Ticker  string          `json:"ticker"`
}

// MarketHistoryRequest is described here:
// https://trading-api.readme.io/reference/getmarkethistory.
type MarketHistoryRequest struct {
	CursorRequest
	MinTS timestamp `json:"min_ts,omitempty"`
	MaxTS timestamp `json:"max_ts,omitempty"`
}

func (c *Client) MarketHistory(
	ctx context.Context,
	ticker string,
	req MarketHistoryRequest,
) (*MarketHistoryResponse, error) {
	var resp MarketHistoryResponse

	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     fmt.Sprintf("markets/%s/history", ticker),
		QueryParams:  req,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// MarketOrderBook is described here:
// https://trading-api.readme.io/reference/getmarketorderbook.
func (c *Client) MarketOrderBook(ctx context.Context, ticker string) (*OrderBook, error) {
	var resp struct {
		OrderBook OrderBook `json:"orderbook"`
	}
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     fmt.Sprintf("markets/%s/orderbook/?depth=100", ticker),
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp.OrderBook, nil
}

// Series is described here:
// https://trading-api.readme.io/reference/getseries.
type Series struct {
	Frequency string `json:"frequency"`
	Ticker    string `json:"ticker"`
	Title     string `json:"title"`
}

// Series is described here:
// https://trading-api.readme.io/reference/getseries.
func (c *Client) Series(ctx context.Context, seriesTicker string) (*Series, error) {
	var resp struct {
		Series Series `json:"series"`
	}
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     fmt.Sprintf("series/%s", seriesTicker),
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp.Series, nil
}
