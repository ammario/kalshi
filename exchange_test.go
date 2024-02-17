package kalshi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExchangeStatus(t *testing.T) {
	t.Parallel()

	client := testClient(t)

	s, err := client.ExchangeStatus(context.Background())
	require.NoError(t, err)
	// The Demo API never sleeps.
	require.True(t, s.ExchangeActive)
	require.True(t, s.TradingActive)
}

func TestExchangeSchedule(t *testing.T) {
	t.Parallel()

	client := testClient(t)

	_, err := client.ExchangeSchedule(context.Background())

	require.NoError(t, err)

}
