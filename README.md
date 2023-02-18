# kalshi

`kalshi` provides a Go implementation of [the Kalshi API](https://trading-api.readme.io/reference/getting-started).

## Endpoints

### Supported
* Market
  * GetMarkets
  * GetTrades
  * GetMarket
  * GetMarketHistory
  * GetMarketOrderbook
  * GetSeries 
* Exchange
  * GetExchangeStatus 
* Auth
  * Login
  * Logout
* Portfolio
  * GetBalance 
  * GetOrders
  * CreateOrder
  * GetOrder
  * CancelOrder
* Market Data Feed (streaming)

### TODO

* Portfolio
    * BatchCreateOrders
    * BatchCancelOrders
    * DecreaseOrder
    * GetPositions
    * GetPortolioSettlements 