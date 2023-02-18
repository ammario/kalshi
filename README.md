# kalshi
[![Go Reference](https://pkg.go.dev/badge/github.com/ammario/kalshi.svg)](https://pkg.go.dev/github.com/ammario/kalshi)
![Go workflow status](https://github.com/ammario/kalshi/actions/workflows/go.yaml/badge.svg)
[![codecov](https://codecov.io/gh/ammario/kalshi/branch/main/graph/badge.svg?token=1SBIWOG23L)](https://codecov.io/gh/ammario/kalshi)


Package `kalshi` provides a Go implementation of [the Kalshi API](https://trading-api.readme.io/reference/getting-started).

```
go get github.com/ammario/kalshi
```

Supports:
* Streaming market data feed
* All core API endpoints
* Rate-limits
* Cursor-based pagination

## Endpoint Support

### Markets

`kalshi` supports all Market endpoints.

| Endpoint           | Support Status |
| ------------------ | -------------- |
| GetSeries          | ✅              |
| GetEvent           | ✅              |
| GetMarkets         | ✅              |
| GetTrades          | ✅              |
| GetMarket          | ✅              |
| GetMarketHistory   | ✅              |
| GetMarketOrderbook | ✅              |
| GetSeries          | ✅              |

### Exchange
`kalshi` supports all Exchange endpoints.

| Endpoint          | Support Status |
| ----------------- | -------------- |
| GetExchangeStatus | ✅              |

### Auth

`kalshi` supports all Auth endpoints.

| Endpoint | Support Status |
| -------- | -------------- |
| Login    | ✅              |
| Logout   | ✅              |

### Portfolio

`kalshi` has mixed support for Portfolio endpoints.

| Endpoint               | Support Status |
| ---------------------- | -------------- |
| GetBalance             | ✅              |
| GetFills               | ✅              |
| GetOrders              | ✅              |
| CreateOrder            | ✅              |
| GetOrder               | ✅              |
| CancelOrder            | ✅              |
| BatchCreateOrders      | ❌              |
| BatchCancelOrders      | ❌              |
| DecreaseOrder          | ✅              |
| GetPositions           | ✅              |
| GetPortolioSettlements | ✅              |

### Market Data Feed 

[Market Data Feed](https://trading-api.readme.io/reference/introduction) is supported, although it hasn't been thoroughly tested. You may open a feed through `(*Client).Feed()`.