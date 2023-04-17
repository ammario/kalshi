package kalshi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderBook(t *testing.T) {
	t.Parallel()

	book := OrderBook{
		YesBids: OrderBookBids{
			{1, 2500},
			{2, 500},
			{3, 100},
		},
	}

	// No book
	_, ok := book.BestYesOffer(100)
	require.False(t, ok)

	var price Cents

	// Since order is small, should execute at best price.
	price, ok = book.BestNoOffer(10)
	require.True(t, ok)
	require.Equal(t, Cents(97), price)

	// Order too large
	_, ok = book.BestNoOffer(4000)
	require.False(t, ok)

	// Order is large, executes at worst price
	price, ok = book.BestNoOffer(3000)
	require.True(t, ok)
	require.Equal(t, Cents(99), price)

	// Order is large, executes at worst price
	price, ok = book.BestNoOffer(3000)
	require.True(t, ok)
	require.Equal(t, Cents(99), price)

	price, ok = book.BestNoOffer(600)
	require.True(t, ok)
	require.Equal(t, Cents(98), price)

	liquidity := book.NoLiquidity()
	require.Equal(t, Cents(306200), liquidity)

	liquidity = book.YesLiquidity()
	require.Equal(t, Cents(0), liquidity)

	offers := book.NoTotalOffers()
	require.Equal(t, 3100, offers)

	offers = book.YesTotalOffers()
	require.Equal(t, 0, offers)

	offers = book.NoOffersUnderLimit(Cents(98))
	require.Equal(t, 600, offers)

	offers = book.YesOffersUnderLimit(Cents(0))
	require.Equal(t, 0, offers)
}

func TestParseOrderBook(t *testing.T) {
	t.Parallel()
	var book OrderBook
	str := `{"yes": [[19, 132], [18, 58]], "no": []}`

	err := json.Unmarshal([]byte(str), &book)
	require.NoError(t, err)
	require.Equal(t, OrderBook{
		YesBids: OrderBookBids{
			{Cents(19), 132},
			{Cents(18), 58},
		},
		NoBids: OrderBookBids{},
	}, book)
}
