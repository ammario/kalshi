package kalshi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/google/go-querystring/query"
	"golang.org/x/time/rate"
)

const (
	APIDemoURL = "https://demo-api.kalshi.co/trade-api/v2/"
	APIProdURL = "https://trading-api.kalshi.com/trade-api/v2/"
)

// Cents is a safety type to prevent dollars from being accidently passed into
// the API.
type Cents = int

// Client must be instantiated via New.
type Client struct {
	// BaseURL is one of APIDemoURL or APIProdURL.
	BaseURL string

	// See https://trading-api.readme.io/reference/tiers-and-rate-limits
	Ratelimit  *rate.Limiter
	httpClient *http.Client
}

type CursorResponse struct {
	Cursor string `json:"cursor"`
}

type CursorRequest struct {
	Limit  int    `url:"limit,omitempty"`
	Cursor string `url:"cursor,omitempty"`
}

type request struct {
	CursorRequest
	Method       string
	Endpoint     string
	QueryParams  any
	JSONRequest  any
	JSONResponse any
}

func jsonRequestHeaders(
	ctx context.Context,
	client *http.Client,
	headers http.Header,
	method string, reqURL string,
	jsonReq interface{}, jsonResp interface{},
) error {
	reqBodyByt, err := json.Marshal(jsonReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, reqURL, bytes.NewReader(reqBodyByt))
	if err != nil {
		return err
	}
	if headers != nil {
		req.Header = headers
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBodyByt, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	reqDump, err := httputil.DumpRequest(req, false)
	if err != nil {
		return err
	}

	respDump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	dumpErr := fmt.Sprintf("Request\n%s%s\nResponse\n%s%s",
		reqDump,
		reqBodyByt,
		respDump,
		respBodyByt,
	)

	if resp.StatusCode >= 400 {
		return fmt.Errorf(
			"unexpected status: %s\n%s",
			resp.Status,
			dumpErr,
		)
	} else if os.Getenv("KALSHI_HTTP_DEBUG") != "" {
		fmt.Printf("REQUEST DUMP\n%s\n", dumpErr)
	}

	if client.Jar != nil {
		u, err := url.Parse(reqURL)
		if err != nil {
			return err
		}
		client.Jar.SetCookies(u, resp.Cookies())
	}

	if jsonResp != nil {
		err = json.Unmarshal(respBodyByt, jsonResp)
		if err != nil {
			return fmt.Errorf("unmarshal: %w\n%s", err, dumpErr)
		}
	}
	return nil
}

func (c *Client) request(
	ctx context.Context, r request,
) error {
	err := c.Ratelimit.Wait(ctx)
	if err != nil {
		return err
	}

	u, err := url.Parse(c.BaseURL + r.Endpoint)
	if err != nil {
		return err
	}

	if r.QueryParams != nil {
		v, err := query.Values(r.QueryParams)
		if err != nil {
			return err
		}
		u.RawQuery = v.Encode()
	}

	return jsonRequestHeaders(
		ctx,
		c.httpClient,
		nil,
		r.Method,
		u.String(), r.JSONRequest, r.JSONResponse,
	)
}

// Timestamp represents a POSIX Timestamp in seconds.
type Timestamp time.Time

func (t Timestamp) Time() time.Time {
	return time.Time(t)
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	i, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}
	*t = Timestamp(time.Unix(int64(i), 0))
	return nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(time.Time(t).UTC().Unix()))), nil
}

// New creates a new Kalshi client. Login must be called to authenticate the
// the client before any other request.
func New(baseURL string) *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	c := &Client{
		httpClient: &http.Client{
			Jar: jar,
		},
		BaseURL: baseURL,
		// See https://trading-api.readme.io/reference/tiers-and-rate-limits.
		// Default to Basic access.
		Ratelimit: rate.NewLimiter(rate.Every(time.Second), 10),
	}

	return c
}

// Time is a time.Time that tolerates additional '"' characters.
// Kalshi API endpoints use both RFC3339 and POSIX
// timestamps.
type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(b []byte) error {
	if len(bytes.Trim(b, "\"")) == 0 {
		return nil
	}
	err := t.Time.UnmarshalJSON(b)
	if err != nil {
		return fmt.Errorf("%v: %w", len(b), err)
	}
	return nil
}

// Side is either Yes or No.
type Side string

const (
	Yes Side = "yes"
	No  Side = "no"
)

// SideBool turns a Yes bool into a Side.
func SideBool(yes bool) Side {
	if yes {
		return Yes
	}
	return No
}
