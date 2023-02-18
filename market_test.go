package kalshi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvents(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)

	t.Run("NoOptions", func(t *testing.T) {
		t.Parallel()
		resp, err := client.Events(ctx, EventsRequest{})
		require.NoError(t, err)
		t.Logf("got %v events", len(resp.Events))
		require.Greater(t, len(resp.Events), 100)
	})

	t.Run("SeriesTicker", func(t *testing.T) {
		t.Parallel()
		resp, err := client.Events(ctx, EventsRequest{
			SeriesTicker: "GTEMP",
		})
		require.NoError(t, err)
		require.Greater(t, len(resp.Events), 1)

		t.Run("GetEvent", func(t *testing.T) {
			t.Parallel()
			ev, err := client.Event(ctx, resp.Events[0].EventTicker)
			require.NoError(t, err)
			require.NotZero(t, ev)
		})
	})
}

func TestTrades(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)

	market := highestVolumeMarkets(ctx, t, client)[0]

	_, err := client.Trades(ctx, TradesRequest{
		Ticker: market.Ticker,
	})
	require.NoError(t, err)
	// It's very possible that there are no trades on a test market.
}

func TestMarkets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client := testClient(t)

	t.Run("NoOptions", func(t *testing.T) {
		t.Parallel()
		resp, err := client.Markets(ctx, MarketsRequest{})
		require.NoError(t, err)
		// 100 is the maximum default limit.
		require.Len(t, resp.Markets, 100)
		require.NotEmpty(t, resp.Cursor)
	})

	t.Run("GTEMP", func(t *testing.T) {
		t.Parallel()
		resp, err := client.Markets(ctx, MarketsRequest{
			SeriesTicker: "GTEMP",
			MinCloseTs:   int(time.Now().Unix()),
		})
		require.NoError(t, err)
		require.Equal(t, len(resp.Markets), 1)

		testMarket := resp.Markets[0]

		t.Run("Market", func(t *testing.T) {
			t.Parallel()
			market, err := client.Market(ctx, testMarket.Ticker)
			require.NoError(t, err)
			require.NotZero(t, market)
		})

		t.Run("MarketHistory", func(t *testing.T) {
			t.Parallel()
			resp, err := client.MarketHistory(ctx, testMarket.Ticker, MarketHistoryRequest{})
			require.NoError(t, err)
			require.NotZero(t, resp)
		})
	})
}

func TestSeries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := testClient(t)

	resp, err := client.Series(ctx, "NASDAQ100")
	require.NoError(t, err)
	require.NotEmpty(t, resp)
}
