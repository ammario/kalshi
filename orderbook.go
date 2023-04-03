package kalshi

import (
	"encoding/json"
	"fmt"
	"math"
)

type OrderBookSide []OrderBookBid

type OrderBookBid struct {
	Price    Cents
	Quantity int
}

func (o OrderBookBid) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%d,%d]", o.Price, o.Quantity)), nil
}

func (o *OrderBookBid) UnmarshalJSON(b []byte) error {
	var raw [2]int
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	*o = OrderBookBid{
		Price:    Cents(raw[0]),
		Quantity: raw[1],
	}
	return nil
}

// BestPrice returns the best price for average execution.
func (b OrderBookSide) BestPrice(wantQuantity int) (Cents, bool) {
	var (
		foundQuantity int
		weightedCum   int
		price         Cents
	)

	// The best priced options are at the end of the book.
	// Range in reverse
	for i := len(b) - 1; i >= 0; i-- {
		line := b[i]
		price = Cents(100) - line.Price
		quantity := line.Quantity

		// If we're above wantQuatity, we reduce the amount we're going to
		// take.
		if rem := (quantity + foundQuantity) - wantQuantity; rem > 0 {
			quantity -= rem
		}

		foundQuantity += quantity
		weightedCum += quantity * int(price)

		if foundQuantity == wantQuantity {
			// We round up to be conservative.
			return Cents(math.Round(float64(weightedCum) / float64(wantQuantity))), true
		} else if foundQuantity > wantQuantity {
			panic(fmt.Sprintf("%+v %+v", foundQuantity, wantQuantity))
		}
	}
	return -1, false
}

// OrderBook is a snapshot of the order book.
// It is described here:
// https://trading-api.readme.io/reference/getmarketorderbook.
type OrderBook struct {
	YesBids OrderBookSide `json:"yes"`
	NoBids  OrderBookSide `json:"no"`
}
