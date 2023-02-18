package kalshi

import (
	"testing"

	"github.com/ammario/arbalshi/currency"
	"github.com/stretchr/testify/require"
)

func TestOrderBook(t *testing.T) {
	book := OrderBook{
		Yes: OrderBookDirection{
			{1, 2500},
			{2, 500},
			{3, 100},
		},
	}

	// No book
	_, ok := book.No.BestPrice(100)
	require.False(t, ok)

	var (
		price currency.Amount
	)

	// Since order is small, should execute at best price.
	price, ok = book.Yes.BestPrice(10)
	require.True(t, ok)
	require.Equal(t, currency.Make(0, 97), price)

	// Order too large
	_, ok = book.Yes.BestPrice(4000)
	require.False(t, ok)

	// Order is large, executes at worst price
	price, ok = book.Yes.BestPrice(3000)
	require.True(t, ok)
	require.Equal(t, currency.Make(0, 99), price)

	// Order is large, executes at worst price
	price, ok = book.Yes.BestPrice(3000)
	require.True(t, ok)
	require.Equal(t, currency.Make(0, 99), price)

	// Order is mid-size, executes at median price.
	price, ok = book.Yes.BestPrice(650)
	require.True(t, ok)
	require.Equal(t, currency.Make(0, 98), price)
}
