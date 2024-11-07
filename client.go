package kalshi

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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

type Cents int

func (c Cents) String() string {
	dollars := float32(c) / 100
	return fmt.Sprintf("$%.2f", dollars)
}

type APIKey struct {
	ID  string
	Key *rsa.PrivateKey
}

func LoadAPIKey(apiKeyID, path string) (*APIKey, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Parse PEM encoded RSA private key
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &APIKey{ID: apiKeyID, Key: rsaKey}, nil
}

// Client must be instantiated via New.
type Client struct {
	// BaseURL is one of APIDemoURL or APIProdURL.
	BaseURL string

	// See https://trading-api.readme.io/reference/tiers-and-rate-limits.
	WriteRatelimit *rate.Limiter
	ReadRateLimit  *rate.Limiter

	// APIKey is optional if you use the Login method.
	// As of 2024-11-07, Login-based auth is not working and returning
	// 403 Forbidden?
	APIKey *APIKey

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

func (c *Client) signRequest(req *http.Request) error {
	if c.APIKey == nil {
		return nil
	}

	timestamp := time.Now().UnixMilli()
	payload := fmt.Sprintf("%d%s%s", timestamp, req.Method, req.URL.Path)

	hashed := crypto.SHA256.New()
	hashed.Write([]byte(payload))
	signature, err := rsa.SignPSS(
		rand.Reader,
		c.APIKey.Key,
		crypto.SHA256,
		hashed.Sum(nil),
		&rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
		})
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	req.Header.Set("KALSHI-ACCESS-KEY", c.APIKey.ID)
	req.Header.Set("KALSHI-ACCESS-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("KALSHI-ACCESS-SIGNATURE", base64.StdEncoding.EncodeToString(signature))

	return nil
}

func (c *Client) jsonRequestHeaders(
	ctx context.Context,
	client *http.Client,
	headers http.Header,
	method string, reqURL string,
	jsonReq any, jsonResp any,
) error {
	var (
		reqBodyReader io.Reader
		reqBodyBytes  []byte
	)
	if jsonReq != nil {
		var err error
		reqBodyBytes, err = json.Marshal(jsonReq)
		if err != nil {
			return err
		}
		reqBodyReader = bytes.NewReader(reqBodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBodyReader)
	if err != nil {
		return err
	}
	if headers != nil {
		req.Header = headers
	}

	if err := c.signRequest(req); err != nil {
		return fmt.Errorf("sign request: %w", err)
	}

	if req.Method == "POST" || req.Method == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ammario/kalshi-go")

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
	dumpErr := fmt.Sprintf("Request %s\n%s%s\nResponse\n%s%s",
		reqURL,
		reqDump,
		reqBodyBytes,
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

	// Do not block via Wait! Trades have to be
	// fast to be meaningful!
	if r.Method == "GET" {
		if !c.ReadRateLimit.Allow() {
			return fmt.Errorf("read ratelimit exceeded")
		}
	} else {
		if !c.WriteRatelimit.Allow() {
			return fmt.Errorf("write ratelimit exceeded")
		}
	}

	return c.jsonRequestHeaders(
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

func basicRateLimit() *rate.Limiter {
	return rate.NewLimiter(rate.Every(time.Second), 10)
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
		WriteRatelimit: basicRateLimit(),
		ReadRateLimit:  basicRateLimit(),
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
