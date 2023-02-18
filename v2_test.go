package kalshi

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/ammario/arbalshi/httpapi"
	"github.com/stretchr/testify/require"
)

func Test_V2Markets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, err := NewFromEnv(ctx)
	require.NoError(t, err)

	t.Run("NoOptions", func(t *testing.T) {
		t.Parallel()
		resp, err := client.V2().Markets(ctx, GetMarketsOptions{})
		require.NoError(t, err)
		// 100 is the maximum default limit.
		require.Len(t, resp.Markets, 100)
		require.NotEmpty(t, resp.Cursor)
	})

	t.Run("INX", func(t *testing.T) {
		t.Parallel()
		resp, err := client.V2().Markets(ctx, GetMarketsOptions{
			SeriesTicker: "INX",
			MaxCloseTs:   int(time.Now().AddDate(0, 0, 7).Unix()),
			MinCloseTs:   int(time.Now().Unix()),
		})
		require.NoError(t, err)
		require.Greater(t, len(resp.Markets), 10)
		require.Less(t, len(resp.Markets), 50)
	})
}

func Test_V2Balance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	clientv1, err := NewFromEnv(ctx)
	require.NoError(t, err)
	client := clientv1.V2()

	b, err := client.Balance(ctx)
	require.NoError(t, err)
	require.Greater(t, b, 0)
	t.Logf("balance: %v", b)
}

func Test_V2Order(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	clientv1, err := NewFromEnv(ctx)
	require.NoError(t, err)
	client := clientv1.V2()

	exchangeStatus, err := client.ExchangeStatus(ctx)
	require.NoError(t, err)

	require.True(t, exchangeStatus.ExchangeActive)
	require.True(t, exchangeStatus.TradingActive)

	resp, err := client.Markets(ctx, GetMarketsOptions{
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
			Expiration: httpapi.Timestamp(time.Now().Add(time.Minute)),
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
			Expiration: httpapi.Timestamp(time.Now().Add(time.Minute)),
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
