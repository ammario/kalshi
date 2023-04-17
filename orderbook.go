package kalshi

import (
	"encoding/json"
	"fmt"
)

type OrderBookBids []OrderBookBid

// OrderBookBid represents the aggregate quantity of all resting Bids at a given price.
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

// YesLiquidity returns the total sum required to buy all available
// Yes contracts on the market.
func (b OrderBook) YesLiquidity() Cents {
	return b.NoBids.liquidity()
}

// NoLiquidity returns the total sum required to buy all available
// No contracts on the market.
func (b OrderBook) NoLiquidity() Cents {
	return b.YesBids.liquidity()
}

// YesTotalOffers is the quantity of total Yes contracts available to be taken.
func (b OrderBook) YesTotalOffers() int {
	return b.NoBids.totalOffers()
}

// NoTotalOffers is the quantity of total No contracts available to be taken.
func (b OrderBook) NoTotalOffers() int {
	return b.YesBids.totalOffers()
}

// liquidity is in an internal method that calculates the liquidity a slice of bids
// provides to takers on the opposite side of the market.
func (b OrderBookBids) liquidity() Cents {
	liquidity := Cents(0)
	for i := 0; i < len(b); i++ {
		liquidity += Cents(b[i].Quantity * int(100-b[i].Price))
	}
	return liquidity
}

// totalOffers sums the quantity of contracts available to takers on the
// opposite side of the market.
func (b OrderBookBids) totalOffers() int {
	total := 0
	for i := 0; i < len(b); i++ {
		total += b[i].Quantity
	}
	return total
}

// offersUnderLimit sums the quantity of contracts available to takers on the
// opposite side of the market at a price less than or equal to the given limit.
func (b OrderBookBids) offersUnderLimit(limitPrice Cents) int {
	quantity := 0
	for i := len(b) - 1; i >= 0; i-- {
		if 100-b[i].Price > limitPrice {
			return quantity
		}
		quantity += b[i].Quantity
	}
	return quantity
}

// YesOffersUnderLimit is the quantity of Yes contracts available to be taken
// at a price less than or equal to the given limit.
func (b OrderBook) YesOffersUnderLimit(limit Cents) int {
	return b.NoBids.offersUnderLimit(limit)
}

// NoOffersUnderLimit is the quantity of No contracts available to be taken
// at a price less than or equal to the given limit.
func (b OrderBook) NoOffersUnderLimit(limit Cents) int {
	return b.YesBids.offersUnderLimit(limit)
}

// BestYesOffer returns the best average
// asking price for Yes contracts given a desired quantity.
func (b OrderBook) BestYesOffer(quantity int) (Cents, bool) {
	return b.NoBids.bestPrice(quantity)
}

// BestNoOffer returns the best average
// asking price for No contracts given a desired quantity.
func (b OrderBook) BestNoOffer(quantity int) (Cents, bool) {
	return b.YesBids.bestPrice(quantity)
}

// bestPrice returns the best average asking price that a slice of bids
// provides to the opposite side of the market.
func (b OrderBookBids) bestPrice(wantQuantity int) (Cents, bool) {
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
			return Cents(conservativeRound(float64(weightedCum) / float64(wantQuantity))), true
		} else if foundQuantity > wantQuantity {
			panic(fmt.Sprintf("%+v %+v", foundQuantity, wantQuantity))
		}
	}
	return -1, false
}

func conservativeRound(a float64) int {
	down := int(a)
	if a-float64(down) > 0 {
		return down + 1
	}
	return down
}

// OrderBook is a snapshot of the order book.
//
// Make sure you understand the market structure before using this struct.
// A central feature of the Kalshi contract model is that a No bid corresponds
// to a Yes ask of the complementary price and vice versa. That is, a No bid at 40 cents
// is equivalent to a Yes ask at 60 cents. This OrderBook type is a
// a list of bids on either side.
//
// Detailed documentation can be found here:
// https://trading-api.readme.io/reference/getmarketorderbook.
type OrderBook struct {
	YesBids OrderBookBids `json:"yes"`
	NoBids  OrderBookBids `json:"no"`
}
