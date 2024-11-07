package kalshi

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

var rateLimit = rate.NewLimiter(rate.Every(time.Second), 10-1)

func testClient(t *testing.T) *Client {
	const (
		apiKeyIDEnv   = "KALSHI_API_KEY_ID"
		apiKeyPathEnv = "KALSHI_API_KEY_PATH"
	)

	ctx := context.Background()

	apiKeyID, ok := os.LookupEnv(apiKeyIDEnv)
	if !ok {
		t.Fatalf("no $%s provided", apiKeyIDEnv)
	}

	apiKeyPath, ok := os.LookupEnv(apiKeyPathEnv)
	if !ok {
		t.Fatalf("no $%s provided", apiKeyPathEnv)
	}

	apiKey, err := LoadAPIKey(apiKeyID, apiKeyPath)
	require.NoError(t, err)

	c := New(APIDemoURL)
	c.WriteRatelimit = rateLimit
	c.APIKey = apiKey

	require.NoError(t, err)
	t.Cleanup(func() {
		// Logout will fail during the Logout test.
		_ = c.Logout(ctx)
	})
	return c
}
