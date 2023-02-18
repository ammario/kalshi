package kalshi

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"golang.org/x/time/rate"
)

type kalshiLoginRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

type kalshiLoginResponse struct {
	Token  string `json:"token,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

func NewFromEnv(ctx context.Context) (*V2Client, error) {
	const (
		emailEnv = "KALSHI_EMAIL"
		passEnv  = "KALSHI_PASSWORD"
	)

	email, ok := os.LookupEnv(emailEnv)
	if !ok {
		return nil, fmt.Errorf("no $%s provided", emailEnv)
	}
	password, ok := os.LookupEnv(passEnv)
	if !ok {
		return nil, fmt.Errorf("no $%s provided", passEnv)
	}
	return New(ctx, email, password)
}

func New(ctx context.Context, email, password string) (*V2Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	c := &V2Client{
		httpClient: &http.Client{
			Jar: jar,
		},
		// See https://trading-api.readme.io/reference/tiers-and-rate-limits.
		Ratelimit: rate.NewLimiter(rate.Every(time.Second*10), (10 - 1)),
	}

	var resp kalshiLoginResponse
	err = c.request(ctx, v2APIRequest{
		Method:   "POST",
		Endpoint: "login",
		JSONRequest: kalshiLoginRequest{
			Email:    email,
			Password: password,
		},
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	c.userID = resp.UserID
	return c, nil
}

type RulebookVariables struct {
	AboveBelowBetween string `json:"Above/Below/Between"`
	CapStrike         string `json:"Cap_strike"`
	Date              string `json:"Date"`
	FloorStrike       string `json:"Floor_strike"`
	Value             string `json:"Value"`
	ContractTicker    string `json:"contract_ticker"`
}

type OrderBookDirection [][2]int

// BestPrice returns the best price for average execution.
func (b OrderBookDirection) BestPrice(wantQuantity int) (Cents, bool) {
	var (
		foundQuantity int
		weightedCum   int
		price         int
	)

	// The best priced options are at the end of the book.
	// Range in reverse
	for i := len(b) - 1; i >= 0; i-- {
		line := b[i]
		price = 100 - line[0]
		quantity := line[1]

		// If we're above wantQuatity, we reduce the amount we're going to
		// take.
		if rem := (quantity + foundQuantity) - wantQuantity; rem > 0 {
			quantity -= rem
		}

		foundQuantity += quantity
		weightedCum += quantity * price

		if foundQuantity == wantQuantity {
			// We round up to be conservative.
			return Cents(math.Round(float64(weightedCum) / float64(wantQuantity))), true

		} else if foundQuantity > wantQuantity {
			panic(fmt.Sprintf("%+v %+v", foundQuantity, wantQuantity))
		}
	}
	return -1, false
}

type OrderBook struct {
	Yes OrderBookDirection `json:"yes"`
	No  OrderBookDirection `json:"no"`
}

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(b []byte) error {
	if len(bytes.Trim(b, "\"")) == 0 {
		return nil
	}
	err := t.Time.UnmarshalJSON(b)
	if err != nil {
		return fmt.Errorf("%v: %w", len(b), err)
	}
	return nil
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
type Side string

const (
	Yes Side = "yes"
	No  Side = "no"
)

// SideBool turns a Yes bool into a Side.
func SideBool(yes bool) Side {
	if yes {
		return Yes
	}
	return No
}

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

func (c *V1Client) AllMarketPositions(ctx context.Context) (
	[]MarketPosition, error,
) {
	var resp struct {
		MarketPositions []MarketPosition `json:"market_positions"`
	}

	err := c.requestv1(ctx, "GET", "users/"+c.userID+"/positions", nil, &resp)
	return resp.MarketPositions, err
}
