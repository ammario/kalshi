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

// SeriesResponse is described here:
// https://trading-api.readme.io/reference/getevents.
type SeriesResponse struct {
	CursorResponse
	Events []Event `json:"events"`
}

// SeriesRequest is described here:
// https://trading-api.readme.io/reference/getevents.
type SeriesRequest struct {
	CursorRequest
	// Status is one of "open", "closed", or "settled".
	Status       string `url:"status,omitempty"`
	SeriesTicker string `url:"series_ticker,omitempty"`
}

// GetSeries is described here:
// https://trading-api.readme.io/reference/getevents.
func (c *Client) GetSeries(ctx context.Context, req SeriesRequest) {

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
	opts GetMarketsRequest,
) (*MarketsResponse, error) {
	var resp MarketsResponse

	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "markets",
		QueryParams:  opts,
		JSONRequest:  nil,
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
