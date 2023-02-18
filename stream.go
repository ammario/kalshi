package kalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Stream struct {
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

func (s *Stream) sendCommand(ctx context.Context, c command) error {
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
		MarketID string             `json:"market_id"`
		Yes      OrderBookDirection `json:"yes"`
		No       OrderBookDirection `json:"no"`
	} `json:"msg"`
}

type orderBookDelta struct {
	subscriptionMessageHeader
	Msg struct {
		MarketID string `json:"market_id"`
		Price    int    `json:"price"`
		Delta    int    `json:"delta"`
		Side     string `json:"side"`
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
	Yes      map[int]int
	No       map[int]int
}

func makeOrderBookStreamState(marketID string) orderBookStreamState {
	return orderBookStreamState{
		MarketID: marketID,
		Yes:      make(map[int]int),
		No:       make(map[int]int),
	}
}

// sortOrderBook performs an in-place sort of OrderBook.
func sortOrderBookDirection(dir OrderBookDirection) {
	sort.Slice(dir, func(i, j int) bool {
		return dir[i][0] < dir[j][0]
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

	for _, v := range book.Yes {
		o.Yes[v[0]] = v[1]
	}

	for _, v := range book.No {
		o.No[v[0]] = v[1]
	}
}

// OrderBook returns a canonical OrderBook from the stream state.
func (o *orderBookStreamState) OrderBook() *StreamOrderBook {
	ob := StreamOrderBook{
		LoadedAt: time.Now(),
		MarketID: o.MarketID,
	}

	for k, v := range o.Yes {
		ob.Yes = append(ob.Yes, [2]int{k, v})
	}
	for k, v := range o.No {
		ob.No = append(ob.No, [2]int{k, v})
	}

	sortOrderBookDirection(ob.No)
	sortOrderBookDirection(ob.Yes)

	return &ob
}

func (o *orderBookStreamState) ApplyDelta(side string, price, delta int) error {
	var dir map[int]int
	if side == "yes" {
		dir = o.Yes
	} else if side == "no" {
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
func (s *Stream) Book(ctx context.Context, marketTicker string, feed chan<- *StreamOrderBook) error {
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
				Yes: snapshot.Msg.Yes,
				No:  snapshot.Msg.No,
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

func (f *Stream) Close() error {
	return f.c.Close(websocket.StatusNormalClosure, "")
}

func (c *V2Client) Stream(ctx context.Context) (*Stream, error) {
	conn, resp, err := websocket.Dial(ctx,
		"wss://trading-api.kalshi.com/trade-api/ws/v2",
		&websocket.DialOptions{
			HTTPClient: c.v1.httpClient,
		},
	)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("websocket refused: %v", resp.Status)
	}

	return &Stream{c: conn}, nil
}
