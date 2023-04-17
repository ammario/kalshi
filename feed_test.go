package kalshi

import (
	"context"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_orderBookStreamState(t *testing.T) {
	t.Parallel()

	sob := makeOrderBookStreamState("duh")

	requireBook := func(wantYes []OrderBookBid, wantNo []OrderBookBid) {
		book := sob.OrderBook()
		require.WithinDuration(t, time.Now(), book.LoadedAt, time.Millisecond*10)
		require.EqualValues(t, wantYes, book.YesBids, "yes")
		require.EqualValues(t, wantNo, book.NoBids, "no")
	}

	// Add entry
	err := sob.ApplyDelta("yes", 10, 20)
	require.NoError(t, err)

	requireBook([]OrderBookBid{
		{10, 20},
	}, nil)

	// Delete entry
	err = sob.ApplyDelta("yes", 10, -20)
	require.NoError(t, err)

	requireBook(nil, nil)

	// Add entry
	err = sob.ApplyDelta("yes", 10, 20)
	require.NoError(t, err)

	err = sob.ApplyDelta("no", 11, 21)
	require.NoError(t, err)

	requireBook([]OrderBookBid{
		{10, 20},
	}, []OrderBookBid{
		{11, 21},
	})

	// Entry sorts
	err = sob.ApplyDelta("yes", 9, 20)
	require.NoError(t, err)

	err = sob.ApplyDelta("no", 10, 21)
	require.NoError(t, err)

	requireBook([]OrderBookBid{
		{9, 20},
		{10, 20},
	}, []OrderBookBid{
		{10, 21},
		{11, 21},
	})

	// Load book
	sob.LoadBook(OrderBook{
		YesBids: []OrderBookBid{
			{19, 8},
		},
		NoBids: []OrderBookBid{
			{20, 9},
		},
	})
	requireBook([]OrderBookBid{
		{19, 8},
	}, []OrderBookBid{
		{20, 9},
	})
}

func highestVolumeMarkets(ctx context.Context, t *testing.T, client *Client) []Market {
	var (
		markets []Market
		cursor  string
	)
	startFind := time.Now()
	for {
		resp, err := client.Markets(ctx, MarketsRequest{
			CursorRequest: CursorRequest{
				Cursor: cursor,
			},
			MinCloseTs: int(time.Now().Unix()),
			// GTEP is arbitrarily chosen to restrict our search space.
			SeriesTicker: "GTEMP",
			Status:       "open",
		})
		require.NoError(t, err)

		// For debug purposes.
		for _, m := range resp.Markets {
			t.Logf("market: %+v", m.Ticker)
		}

		markets = append(markets, resp.Markets...)
		if resp.Cursor != "" {
			cursor = resp.Cursor
			continue
		}
		break
	}
	sort.Slice(markets, func(i, j int) bool {
		// Volume24H may be caused by a single whale, leading to inconsistent
		// results.
		return markets[i].Volume*markets[i].Volume24H > markets[j].Volume*markets[j].Volume24H
	})

	t.Logf("found %v open markets in %v", len(markets), time.Since(startFind))

	return markets
}

func TestFeed(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.SkipNow()
	}

	if os.Getenv("TEST_STREAM") == "" {
		t.Skip("this test is racey and unreliable")
	}

	ctx := context.Background()

	client := testClient(t)

	markets := highestVolumeMarkets(ctx, t, client)

	verifyBook := func(t *testing.T, marketTicker string, gotBook *StreamOrderBook) {
		require.NotNil(t, gotBook)

		var (
			wantBook *OrderBook
			err      error
		)

		t.Logf("verifiying: %v", marketTicker)

		// The polling API can lag behind sometimes.
		assert.Eventually(t, func() bool {
			wantBook, err = client.MarketOrderBook(ctx, marketTicker)
			require.NoError(t, err)

			return reflect.DeepEqual(wantBook.NoBids, gotBook.NoBids) && reflect.DeepEqual(wantBook.YesBids, gotBook.YesBids)
		}, time.Second*10, time.Second)

		// This gives a pretty error.
		require.Equal(t, wantBook.YesBids, gotBook.YesBids, "Yes")
		require.Equal(t, wantBook.NoBids, gotBook.NoBids, "No")
	}

	longStreamOrSkip := func(t *testing.T) {
		testLongStreamDur := os.Getenv("TEST_LONG_STREAM")
		if testLongStreamDur == "" {
			t.Skip("not doing long stream")
		}
		t.Logf("beginning to test long stream")
	}

	t.Run("Simple", func(t *testing.T) {
		t.Parallel()

		m := markets[0]
		t.Logf("targeting market %v", m.Ticker)

		t.Logf("highest volume market: %+v %v (24h) %v (all time)", m.Ticker, m.Volume24H, m.Volume)

		s, err := client.OpenFeed(ctx)
		require.NoError(t, err)
		defer s.Close()

		var (
			bookErr error
			bookCh  = make(chan *StreamOrderBook)
		)
		go func() {
			bookErr = s.Book(ctx, m.Ticker, bookCh)
			close(bookCh)
		}()

		book, ok := <-bookCh
		require.True(t, ok, "book error: %+v", bookErr)

		verifyBook(t, m.Ticker, book)

		require.NoError(t, bookErr)

		longStreamOrSkip(t)

		const wantUpdates = 5
		var (
			recheck     = time.NewTicker(time.Second * 30)
			updateCount = 0
		)
		for {
			select {
			case <-recheck.C:
				// If we're not seeing an update, the book shouldn't be changing
				// under our feet.
				verifyBook(t, m.Ticker, book)
				if updateCount >= wantUpdates {
					return
				}
			case book, ok = <-bookCh:
				if !ok {
					t.Fatal("book channel closed")
				}
				t.Logf("got book update! (%v/%v)", updateCount+1, wantUpdates)
				// It can take a moment for the polling API to update its book.
				recheck.Reset(time.Second * 5)
				updateCount++
			}
		}
	})
}
