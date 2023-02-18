package kalshi

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// timestmap represents a UNIX timestamp in seconds suitable for the Kalshi
// JSON HTTP API.
type timestamp time.Time

func (t timestamp) Time() time.Time {
	return time.Time(t)
}

func (t *timestamp) UnmarshalJSON(byt []byte) error {
	secs, err := strconv.Atoi(string(byt))
	if err != nil {
		return err
	}
	*t = timestamp(time.Unix(int64(secs), 0))
	return nil
}

func (t timestamp) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(time.Time(t).UTC().Unix()))), nil
}

// New creates a new Kalshi client. Login must be called to authenticate the
// the client before any other request.
func New(ctx context.Context, baseURL string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	c := &Client{
		httpClient: &http.Client{
			Jar: jar,
		},
		BaseURL: baseURL,
		// See https://trading-api.readme.io/reference/tiers-and-rate-limits.
		// Default to Basic access.
		Ratelimit: rate.NewLimiter(rate.Every(time.Second*10), (10 - 1)),
	}

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
