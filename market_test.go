package kalshi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMarkets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client := testClient(t)

	t.Run("NoOptions", func(t *testing.T) {
		t.Parallel()
		resp, err := client.Markets(ctx, GetMarketsOptions{})
		require.NoError(t, err)
		// 100 is the maximum default limit.
		require.Len(t, resp.Markets, 100)
		require.NotEmpty(t, resp.Cursor)
	})

	t.Run("INX", func(t *testing.T) {
		t.Parallel()
		resp, err := client.Markets(ctx, GetMarketsOptions{
			SeriesTicker: "INX",
			MaxCloseTs:   int(time.Now().AddDate(0, 0, 7).Unix()),
			MinCloseTs:   int(time.Now().Unix()),
		})
		require.NoError(t, err)
		require.Greater(t, len(resp.Markets), 10)
		require.Less(t, len(resp.Markets), 50)
	})
}
