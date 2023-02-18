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

	requireBook := func(wantYes [][2]int, wantNo [][2]int) {
		book := sob.OrderBook()
		require.WithinDuration(t, time.Now(), book.LoadedAt, time.Millisecond*10)
		require.EqualValues(t, wantYes, book.Yes, "yes")
		require.EqualValues(t, wantNo, book.No, "no")
	}

	// Add entry
	sob.ApplyDelta("yes", 10, 20)
	requireBook([][2]int{
		{10, 20},
	}, nil)

	// Delete entry
	sob.ApplyDelta("yes", 10, -20)
	requireBook(nil, nil)

	// Add entry
	sob.ApplyDelta("yes", 10, 20)
	sob.ApplyDelta("no", 11, 21)
	requireBook([][2]int{
		{10, 20},
	}, [][2]int{
		{11, 21},
	})

	// Entry sorts
	sob.ApplyDelta("yes", 9, 20)
	sob.ApplyDelta("no", 10, 21)
	requireBook([][2]int{
		{9, 20},
		{10, 20},
	}, [][2]int{
		{10, 21},
		{11, 21},
	})

	// Load book
	sob.LoadBook(OrderBook{
		Yes: [][2]int{
			{19, 8},
		},
		No: [][2]int{
			{20, 9},
		},
	})
	requireBook([][2]int{
		{19, 8},
	}, [][2]int{
		{20, 9},
	})
}

func highestVolumeMarkets(ctx context.Context, t *testing.T, client *V2Client) []V2Market {
	var (
		markets []V2Market
		cursor  string
	)
	startFind := time.Now()
	for {
		resp, err := client.Markets(ctx, GetMarketsOptions{
			CursorRequest: CursorRequest{
				Cursor: cursor,
			},
			MinCloseTs: int(time.Now().Unix()),
			// We know from history that FED contains the most popular markets.
			// If we iterated over _all_ of the markets, we'd send a lot
			// of requests, adding unwanted test latency.
			// SeriesTicker: "FED",
			Status: "open",
		})
		require.NoError(t, err)

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

func TestStream(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.SkipNow()
	}

	t.Skip("sus")

	ctx := context.Background()

	client, err := NewFromEnv(ctx)
	require.NoError(t, err)

	markets := highestVolumeMarkets(ctx, t, client)

	verifyBook := func(t *testing.T, marketTicker string, gotBook *StreamOrderBook) {
		var (
			wantBook *OrderBook
			err      error
		)

		t.Logf("verifiying: %v", marketTicker)

		// The polling API can lag behind sometimes.
		assert.Eventually(t, func() bool {
			wantBook, err = client.MarketOrderBook(ctx, marketTicker)
			require.NoError(t, err)

			return reflect.DeepEqual(wantBook.No, gotBook.No) && reflect.DeepEqual(wantBook.Yes, gotBook.Yes)
		}, time.Second*10, time.Second, "%+v")

		// This gives a pretty error.
		require.Equal(t, wantBook.Yes, gotBook.Yes, "Yes")
		require.Equal(t, wantBook.No, gotBook.No, "No")
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

		s, err := client.Stream(ctx)
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
