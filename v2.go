package kalshi

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	APIDemoURL = "https://demo-api.kalshi.co/trade-api/v2/"
	APIProdURL = "https://trading-api.kalshi.com/trade-api/v2/"
)

type GetMarketsOptions struct {
	CursorRequest
	EventTicker  string `url:"event_ticker,omitempty"`
	SeriesTicker string `url:"series_ticker,omitempty"`
	MaxCloseTs   int    `url:"max_close_ts,omitempty"`
	MinCloseTs   int    `url:"min_close_ts,omitempty"`
	// Status is one of "open", "closed", and "settled"
	Status string `url:"status,omitempty"`
}

type V2Market struct {
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

type V2MarketsResponse struct {
	Markets []V2Market
	CursorResponse
}

func (c *Client) Markets(
	ctx context.Context,
	opts GetMarketsOptions,
) (*V2MarketsResponse, error) {
	if opts.Limit == 0 {
		// Binary-searched maximum
		opts.Limit = 100
	}

	v, err := query.Values(opts)
	if err != nil {
		return nil, err
	}

	var resp V2MarketsResponse

	err = c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "markets/?" + v.Encode(),
		JSONRequest:  nil,
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

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
