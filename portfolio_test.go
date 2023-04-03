package kalshi

import (
	"context"
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

	testMarket := highestVolumeMarkets(ctx, t, client)[0]

	book, err := client.MarketOrderBook(ctx, testMarket.Ticker)
	require.NoError(t, err)
	t.Logf("book for %s: %+v", testMarket.Ticker, book)
	if len(book.NoBids) > 0 {
		bestPrice, ok := book.NoBids.BestPrice(1)
		if ok {
			t.Logf("best price: %v", bestPrice)
		}
	}

	orders, err := client.Orders(ctx, OrdersRequest{
		Status: Resting,
		Ticker: testMarket.Ticker,
	})
	require.NoError(t, err)
	t.Logf("orders: %+v", orders)

	t.Run("Market", func(t *testing.T) {
		createReq := CreateOrderRequest{
			Action:     Buy,
			Count:      1,
			Expiration: ExpireAfter(time.Minute),
			Ticker:     testMarket.Ticker,
			BuyMaxCost: 1,
			Type:       MarketOrder,
			Side:       Yes,
		}
		t.Logf("create req: %+v", createReq.String())
		order, err := client.CreateOrder(ctx, createReq)
		require.NoError(t, err)
		t.Logf("created order: %+v", order)
		require.True(t, order.ExpirationTime.After((time.Now())))
		// Market order should execute immediately.
		require.Equal(t, "executed", order.Status)
		t.Run("Fills", func(t *testing.T) {
			t.Skip("Doesn't seem to work?")
			fills, err := client.Fills(ctx, FillsRequest{
				OrderID: order.OrderID,
			})
			require.NoError(t, err)
			require.Len(t, fills.Fills, 1)
			t.Logf("fills: %+v", fills)
		})
	})

	t.Run("Limit", func(t *testing.T) {
		// Testing limit
		createReq := CreateOrderRequest{
			Action:     Buy,
			Count:      2,
			Expiration: ExpireAfter(time.Minute),
			Ticker:     testMarket.Ticker,
			YesPrice:   1,
			Type:       LimitOrder,
			Side:       Yes,
		}
		order, err := client.CreateOrder(ctx, createReq)
		require.NoError(t, err)
		t.Logf("created order: %+v", order)

		t.Run("Order", func(t *testing.T) {
			t.Parallel()
			order, err := client.Order(ctx, order.OrderID)
			require.NoError(t, err)
			require.NotZero(t, order)
		})

		require.Eventually(t, func() bool {
			orders, err = client.Orders(ctx, OrdersRequest{
				Status: Resting,
				Ticker: testMarket.Ticker,
			})
			require.NoError(t, err)
			return len(orders.Orders) > 0
		}, time.Second*5, time.Millisecond*500)

		for _, order := range orders.Orders {
			t.Logf("order: %+v", order)
		}

		t.Run("Decrease", func(t *testing.T) {
			order, err := client.Order(ctx, order.OrderID)
			require.NoError(t, err)
			require.Equal(t, order.RemainingCount, 2)

			order, err = client.DecreaseOrder(ctx, order.OrderID, DecreaseOrderRequest{
				ReduceBy: 1,
			})
			require.NoError(t, err)
			require.Equal(t, order.RemainingCount, 1)
		})

		t.Run("Positions", func(t *testing.T) {
			resp, err := client.Positions(ctx, PositionsRequest{})
			require.NoError(t, err)
			require.Greater(t, len(resp.EventPositions), 0)
			require.Greater(t, len(resp.MarketPositions), 0)
		})

		t.Run("Cancel", func(t *testing.T) {
			order, err := client.CancelOrder(ctx, order.OrderID)
			require.NoError(t, err)
			require.Equal(t, order.Status, "canceled")
		})
	})
}

func TestSettlements(t *testing.T) {
	t.Parallel()

	client := testClient(t)

	ctx := context.Background()

	_, err := client.Settlements(ctx, SettlementsRequest{})
	require.NoError(t, err)
}
