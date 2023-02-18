package kalshi

import "context"

// ExchangeStatusResponse is described here:
// https://trading-api.readme.io/reference/getexchangestatus.
type ExchangeStatusResponse struct {
	ExchangeActive bool `json:"exchange_active,omitempty"`
	TradingActive  bool `json:"trading_active,omitempty"`
}

// ExchangeStatus is described here:
// https://trading-api.readme.io/reference/getexchangestatus.
func (c *Client) ExchangeStatus(ctx context.Context) (*ExchangeStatusResponse, error) {
	var resp ExchangeStatusResponse
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "exchange/status",
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
