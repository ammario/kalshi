package kalshi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderBook(t *testing.T) {
	t.Parallel()

	book := OrderBook{
		YesBids: OrderBookSide{
			{1, 2500},
			{2, 500},
			{3, 100},
		},
	}

	// No book
	_, ok := book.NoBids.BestPrice(100)
	require.False(t, ok)

	var price Cents

	// Since order is small, should execute at best price.
	price, ok = book.YesBids.BestPrice(10)
	require.True(t, ok)
	require.Equal(t, Cents(97), price)

	// Order too large
	_, ok = book.YesBids.BestPrice(4000)
	require.False(t, ok)

	// Order is large, executes at worst price
	price, ok = book.YesBids.BestPrice(3000)
	require.True(t, ok)
	require.Equal(t, Cents(99), price)

	// Order is large, executes at worst price
	price, ok = book.YesBids.BestPrice(3000)
	require.True(t, ok)
	require.Equal(t, Cents(99), price)

	// Order is mid-size, executes at median price.
	price, ok = book.YesBids.BestPrice(650)
	require.True(t, ok)
	require.Equal(t, Cents(98), price)
}
