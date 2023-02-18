package kalshi

import (
	"fmt"
	"math"
)

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

// OrderBook is a snapshot of the order book.
// It is described here:
// https://trading-api.readme.io/reference/getmarketorderbook.
type OrderBook struct {
	Yes OrderBookDirection `json:"yes"`
	No  OrderBookDirection `json:"no"`
}
