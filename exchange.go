package kalshi

import "context"

// ExchangeStatusResponse is described here:
// https://trading-api.readme.io/reference/getexchangestatus.
type ExchangeStatusResponse struct {
	ExchangeActive bool `json:"exchange_active,omitempty"`
	TradingActive  bool `json:"trading_active,omitempty"`
}

// ExchangeScheduleResponse is described here:
// https://trading-api.readme.io/reference/getexchangeschedule.
type ExchangeScheduleResponse struct {
	Schedule struct {
		StandardHours struct {
			Monday struct {
				OpenTime  string `json:"open_time"`
				CloseTime string `json:"close_time"`
			} `json:"monday"`
			Tuesday struct {
				OpenTime  string `json:"open_time"`
				CloseTime string `json:"close_time"`
			} `json:"tuesday"`
			Wednesday struct {
				OpenTime  string `json:"open_time"`
				CloseTime string `json:"close_time"`
			} `json:"wednesday"`
			Thursday struct {
				OpenTime  string `json:"open_time"`
				CloseTime string `json:"close_time"`
			} `json:"thursday"`
			Friday struct {
				OpenTime  string `json:"open_time"`
				CloseTime string `json:"close_time"`
			} `json:"friday"`
			Saturday struct {
				OpenTime  string `json:"open_time"`
				CloseTime string `json:"close_time"`
			} `json:"saturday"`
			Sunday struct {
				OpenTime  string `json:"open_time"`
				CloseTime string `json:"close_time"`
			} `json:"sunday"`
		} `json:"standard_hours"`
		MaintenanceWindows []struct {
			EndDatetime   string `json:"end_datetime"`
			StartDatetime string `json:"start_datetime"`
		} `json:"maintenance_windows,omitempty"`
	} `json:"schedule"`
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

// ExchangeSchedule is described here:
// https://trading-api.readme.io/reference/getexchangeschedule.
func (c *Client) ExchangeSchedule(ctx context.Context) (*ExchangeScheduleResponse, error) {
	var resp ExchangeScheduleResponse
	err := c.request(ctx, request{
		Method:       "GET",
		Endpoint:     "exchange/schedule",
		JSONResponse: &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
