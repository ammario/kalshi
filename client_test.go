package kalshi

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

var rateLimit = rate.NewLimiter(rate.Every(time.Second*10), 10-1)

func testClient(t *testing.T) *Client {
	const (
		emailEnv = "KALSHI_EMAIL"
		passEnv  = "KALSHI_PASSWORD"
	)

	ctx := context.Background()

	email, ok := os.LookupEnv(emailEnv)
	if !ok {
		t.Fatalf("no $%s provided", emailEnv)
	}
	password, ok := os.LookupEnv(passEnv)
	if !ok {
		t.Fatalf("no $%s provided", passEnv)
	}

	c, err := New(ctx, APIDemoURL)
	require.NoError(t, err)
	c.Ratelimit = rateLimit
	_, err = c.Login(ctx, LoginRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	return c
}
