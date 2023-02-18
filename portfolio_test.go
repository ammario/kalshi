package kalshi

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBalance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client := testClient(t)

	b, err := client.Balance(ctx)
	require.NoError(t, err)
	require.Greater(t, b, 0)
	// Sanity-check
	t.Logf("balance: %v", b)
}

func TestOrder(t *testing.T) {
	t.Parallel()

	client := testClient(t)

	ctx := context.Background()

	exchangeStatus, err := client.ExchangeStatus(ctx)
	require.NoError(t, err)

	// This doesn't really work otherwise.
	require.True(t, exchangeStatus.ExchangeActive)
	require.True(t, exchangeStatus.TradingActive)

	resp, err := client.Markets(ctx, GetMarketsRequest{
		SeriesTicker: "INX",
		MaxCloseTs:   int(time.Now().AddDate(0, 0, 7).Unix()),
		MinCloseTs:   int(time.Now().Unix()),
	})
	require.NoError(t, err)

	// High volume markets will likely not execute my orders at 1 cent...
	sort.Slice(resp.Markets, func(i, j int) bool {
		return resp.Markets[i].Volume24H > resp.Markets[j].Volume24H
	})

	testMarket := resp.Markets[0]

	book, err := client.MarketOrderBook(ctx, testMarket.Ticker)
	require.NoError(t, err)
	t.Logf("book for %s: %+v", testMarket.Ticker, book)
	if len(book.No) > 0 {
		bestPrice, ok := book.No.BestPrice(1)
		if ok {
			t.Logf("best price: %v", bestPrice)
		}
	}

	orders, err := client.Orders(ctx, OrdersRequest{
		Status: "resting",
		Ticker: testMarket.Ticker,
	})
	require.NoError(t, err)
	t.Logf("orders: %+v", orders)

	t.Run("Market", func(t *testing.T) {
		// Order is so cheap that even if it executed we wouldn't care.
		// This should look very similar to the order in the executor.
		createReq := CreateOrderRequest{
			Action:     "buy",
			Count:      1,
			Expiration: timestamp(time.Now().Add(time.Minute)),
			Ticker:     testMarket.Ticker,
			BuyMaxCost: 1,
			Type:       "market",
			Side:       Yes,
		}
		order, err := client.CreateOrder(ctx, createReq)
		require.NoError(t, err)
		t.Logf("created order: %+v", order)
		require.True(t, order.Expiration.After((time.Now())))
		// Market order should execute immediately.
		require.Equal(t, "executed", order.Status)

	})

	t.Run("Limit", func(t *testing.T) {
		// Testing limit
		createReq := CreateOrderRequest{
			Action:     "buy",
			Count:      1,
			Expiration: timestamp(time.Now().Add(time.Minute)),
			Ticker:     testMarket.Ticker,
			YesPrice:   1,
			Type:       "limit",
			Side:       Yes,
		}
		// Check idempotence
		nextOrder, err := client.CreateOrder(ctx, createReq)
		require.NoError(t, err)
		t.Logf("created order: %+v", nextOrder)

		require.Eventually(t, func() bool {
			orders, err = client.Orders(ctx, OrdersRequest{
				Status: "resting",
				Ticker: testMarket.Ticker,
			})
			require.NoError(t, err)
			return len(orders) > 0
		}, time.Second*5, time.Millisecond*500)

		for _, order := range orders {
			t.Logf("order: %+v", order)
		}
	})
}
