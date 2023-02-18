package kalshi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/google/uuid"
	"tailscale.com/tstime/rate"
)

const (
	APIDemoURL = "https://demo-api.kalshi.co/trade-api/v2/"
	APIProdURL = "https://trading-api.kalshi.com/trade-api/v2/"
)

type Client struct {
	httpClient *http.Client

	// BaseURL is one of APIDemoURL or APIProdURL.
	BaseURL string

	// See https://trading-api.readme.io/reference/tiers-and-rate-limits
	Ratelimit *rate.Limiter
	userID    string
}

type CursorResponse struct {
	Cursor string `json:"cursor"`
}

type CursorRequest struct {
	Limit  int    `url:"limit,omitempty"`
	Cursor string `url:"cursor,omitempty"`
}

type request struct {
	CursorRequest
	Method       string
	Endpoint     string
	JSONRequest  any
	JSONResponse any
}

func jsonRequestHeaders(
	ctx context.Context,
	client *http.Client,
	headers http.Header,
	method string, reqURL string,
	jsonReq interface{}, jsonResp interface{},
) error {
	reqBodyByt, err := json.Marshal(jsonReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, reqURL, bytes.NewReader(reqBodyByt))
	if err != nil {
		return err
	}
	if headers != nil {
		req.Header = headers
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBodyByt, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	reqDump, err := httputil.DumpRequest(req, false)
	if err != nil {
		return err
	}

	respDump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	var dumpErr = fmt.Sprintf("Request\n%s%s\nResponse\n%s%s",
		reqDump,
		reqBodyByt,
		respDump,
		respBodyByt,
	)

	if resp.StatusCode >= 400 {
		return fmt.Errorf(
			"unexpected status: %s\n%s",
			resp.Status,
			dumpErr,
		)
	} else if os.Getenv("HTTP_DEBUG") != "" {
		fmt.Printf("REQUEST DUMP\n%s\n", dumpErr)
	}

	if client.Jar != nil {
		u, err := url.Parse(reqURL)
		if err != nil {
			return err
		}
		client.Jar.SetCookies(u, resp.Cookies())
	}

	if jsonResp != nil {
		err = json.Unmarshal(respBodyByt, jsonResp)
		if err != nil {
			return fmt.Errorf("unmarshal: %w\n%s", err, dumpErr)
		}
	}
	return nil
}

func (c *Client) request(
	ctx context.Context, r request,
) error {
	err := c.Ratelimit.Wait(ctx)
	if err != nil {
		return err
	}

	return httpapi.JSONRequest(
		ctx,
		c.httpClient,
		r.Method, baseURLv2+r.Endpoint, r.JSONRequest, r.JSONResponse,
	)
}

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

type CreateOrderRequest struct {
	// Action is either "buy" or "sell"
	Action        string            `json:"action,omitempty"`
	BuyMaxCost    int               `json:"buy_max_cost,omitempty"`
	Count         int               `json:"count,omitempty"`
	Expiration    httpapi.Timestamp `json:"expiration_ts,omitempty"`
	NoPrice       int               `json:"no_price,omitempty"`
	YesPrice      int               `json:"yes_price,omitempty"`
	Ticker        string            `json:"ticker,omitempty"`
	ClientOrderID string            `json:"client_order_id,omitempty"`
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

type OrdersRequest struct {
	Ticker string `url:"ticker,omitempty"`
	Status string `url:"status,omitempty"`
}

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
