package kalshi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	t.Parallel()

	// testClient itself calls Login.
	_ = testClient(t)
}

func TestLogout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := testClient(t)

	_, err := c.Balance(ctx)
	require.NoError(t, err)

	err = c.Logout(ctx)
	require.NoError(t, err)

	_, err = c.Balance(ctx)
	require.Error(t, err)
}
