package kalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Feed is a websocket connection to the Kalshi streaming API.
// Feed is described in more detail here:
// https://trading-api.readme.io/reference/introduction.
// WARNING: Feed has not been thoroughly tested.
type Feed struct {
	c *websocket.Conn
}

type commandParams struct {
	Channels     []string `json:"channels,omitempty"`
	MarketTicker string   `json:"market_ticker,omitempty"`
}
type command struct {
	ID      int           `json:"id,omitempty"`
	Command string        `json:"cmd,omitempty"`
	Params  commandParams `json:"params,omitempty"`
}

func (s *Feed) sendCommand(ctx context.Context, c command) error {
	return wsjson.Write(ctx, s.c, c)
}

type subscribedResponse struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	Msg  struct {
		Channel string `json:"channel"`
		Sid     int    `json:"sid"`
		Msg     string `json:"msg"`
	} `json:"msg"`
}

type orderBookSnapshot struct {
	subscriptionMessageHeader
	Msg struct {
		MarketID string        `json:"market_id"`
		Yes      OrderBookBids `json:"yes"`
		No       OrderBookBids `json:"no"`
	} `json:"msg"`
}

type orderBookDelta struct {
	subscriptionMessageHeader
	Msg struct {
		MarketID string `json:"market_id"`
		Price    Cents  `json:"price"`
		Delta    int    `json:"delta"`
		Side     Side   `json:"side"`
	}
}

type errorMessage struct {
	subscriptionMessageHeader
	Msg struct {
		Code int    `json:"code,omitempty"`
		Msg  string `json:"msg,omitempty"`
	} `json:"msg,omitempty"`
}

type subscriptionMessageHeader struct {
	Type string `json:"type"`
	Sid  int    `json:"sid"`
	Seq  int    `json:"seq"`
}

// orderBookStreamState is kept by Book and serialized into OrderBook.
type orderBookStreamState struct {
	MarketID string
	Yes      map[Cents]int
	No       map[Cents]int
}

func makeOrderBookStreamState(marketID string) orderBookStreamState {
	return orderBookStreamState{
		MarketID: marketID,
		Yes:      make(map[Cents]int),
		No:       make(map[Cents]int),
	}
}

// sortOrderBook performs an in-place sort of OrderBook.
func sortOrderBookDirection(dir OrderBookBids) {
	sort.Slice(dir, func(i, j int) bool {
		return dir[i].Price < dir[j].Price
	})
}

func clear[K comparable, V any](m map[K]V) {
	for k := range m {
		delete(m, k)
	}
}

func (o *orderBookStreamState) LoadBook(book OrderBook) {
	clear(o.Yes)
	clear(o.No)

	for _, v := range book.YesBids {
		o.Yes[v.Price] = v.Quantity
	}

	for _, v := range book.NoBids {
		o.No[v.Price] = v.Quantity
	}
}

// OrderBook returns a canonical OrderBook from the stream state.
func (o *orderBookStreamState) OrderBook() *StreamOrderBook {
	ob := StreamOrderBook{
		LoadedAt: time.Now(),
		MarketID: o.MarketID,
	}

	for k, v := range o.Yes {
		ob.YesBids = append(ob.YesBids, OrderBookBid{Price: k, Quantity: v})
	}
	for k, v := range o.No {
		ob.NoBids = append(ob.NoBids, OrderBookBid{Price: k, Quantity: v})
	}

	sortOrderBookDirection(ob.NoBids)
	sortOrderBookDirection(ob.YesBids)

	return &ob
}

func (o *orderBookStreamState) ApplyDelta(side Side, price Cents, delta int) error {
	var dir map[Cents]int
	if side == Yes {
		dir = o.Yes
	} else if side == No {
		dir = o.No
	} else {
		return fmt.Errorf("unknown side: %v", side)
	}

	current := dir[price]
	current += delta
	if current < 0 {
		return fmt.Errorf("delta when below zero")
	} else if current == 0 {
		delete(dir, price)
	} else {
		dir[price] = current
	}

	return nil
}

// StreamOrderBook is sent by the streaming connection.
type StreamOrderBook struct {
	OrderBook
	LoadedAt time.Time
	// MarketID is included so multiple streaming order books can be multiplexed
	// onto one channel.
	MarketID string
}

// Book instantiates a streaming order book feed for market.
func (s *Feed) Book(ctx context.Context, marketTicker string, feed chan<- *StreamOrderBook) error {
	id := 1
	err := s.sendCommand(ctx, command{
		ID:      1,
		Command: "subscribe",
		Params: commandParams{
			Channels: []string{
				"orderbook_delta",
			},
			MarketTicker: marketTicker,
		},
	})
	if err != nil {
		return err
	}

	var r subscribedResponse
	err = wsjson.Read(ctx, s.c, &r)
	if err != nil {
		return err
	}

	if r.Type != "subscribed" {
		return fmt.Errorf("unexpected message: %+v", r)
	}

	if r.ID != id {
		return fmt.Errorf("unexpected id: %+v", id)
	}

	sid := r.Msg.Sid
	wantSeq := 1

	orderBookState := makeOrderBookStreamState(marketTicker)

	for {
		_, message, err := s.c.Read(ctx)
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		var header subscriptionMessageHeader
		err = json.Unmarshal(message, &header)
		if err != nil {
			return fmt.Errorf("read header: %+v", err)
		}

		if header.Sid != sid {
			return fmt.Errorf("unexpected sid %v", header.Sid)
		}
		if header.Seq != wantSeq {
			return fmt.Errorf("unexpected sequence %v, want %v", header.Seq, wantSeq)
		}
		wantSeq++

		switch header.Type {
		case "orderbook_snapshot":
			var snapshot orderBookSnapshot
			err = json.Unmarshal(message, &snapshot)
			if err != nil {
				return fmt.Errorf("unmarshal snapshot: %w", err)
			}
			ob := OrderBook{
				YesBids: snapshot.Msg.Yes,
				NoBids:  snapshot.Msg.No,
			}
			orderBookState.LoadBook(ob)
			feed <- orderBookState.OrderBook()
		case "orderbook_delta":
			var delta orderBookDelta
			err = json.Unmarshal(message, &delta)
			if err != nil {
				return fmt.Errorf("unmarshal delta: %w", err)
			}
			err = orderBookState.ApplyDelta(
				delta.Msg.Side, delta.Msg.Price, delta.Msg.Delta,
			)
			if err != nil {
				return fmt.Errorf("apply delta: %w", err)
			}
			feed <- orderBookState.OrderBook()
		case "error":
			var errMsg errorMessage
			err = json.Unmarshal(message, &errMsg)
			if err != nil {
				return fmt.Errorf("unmarshal error: %w", err)
			}
			return fmt.Errorf("error message (%v): %v", errMsg.Msg.Code, errMsg.Msg.Msg)
		default:
			return fmt.Errorf("unexpected type %q", header.Type)
		}
	}
}

func (f *Feed) Close() error {
	return f.c.Close(websocket.StatusNormalClosure, "")
}

// OpenFeed creates a new market data streaming connection.
// OpenFeed is described in more detail here:
// https://trading-api.readme.io/reference/introduction.
// WARNING: OpenFeed has not been thoroughly tested.
func (c *Client) OpenFeed(ctx context.Context) (*Feed, error) {
	// Convert BaseURL to a websocket URL.
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", c.BaseURL, err)
	}
	u.Scheme = "wss"
	u.Path = "trade-api/ws/v2"

	conn, resp, err := websocket.Dial(ctx,
		u.String(),
		&websocket.DialOptions{
			HTTPClient: c.httpClient,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("dial %q: %w", u.String(), err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("websocket refused: %v", resp.Status)
	}

	return &Feed{c: conn}, nil
}
