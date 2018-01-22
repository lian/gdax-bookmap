package orderbook

import (
	"fmt"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
	"github.com/lian/gdax-bookmap/util"
)

var CachedInfo map[string]product_info.Info

func init() {
	CachedInfo = map[string]product_info.Info{
		"BTC-USD": product_info.Info{
			Platform:       "Bitstamp",
			DatabaseKey:    "Bitstamp-BTC-USD",
			ID:             "BTC-USD",
			DisplayName:    "BTC-USD",
			BaseCurrency:   "BTC",
			QuoteCurrency:  "USD",
			BaseMinSize:    0,
			BaseMaxSize:    0,
			QuoteIncrement: 0.01,
			FloatFormat:    fmt.Sprintf("%%.%df", util.NumDecPlaces(0.01)),
		},
		"ETH-USD": product_info.Info{
			Platform:       "Bitstamp",
			DatabaseKey:    "Bitstamp-ETH-USD",
			ID:             "ETH-USD",
			DisplayName:    "ETH-USD",
			BaseCurrency:   "ETH",
			QuoteCurrency:  "USD",
			BaseMinSize:    0,
			BaseMaxSize:    0,
			QuoteIncrement: 0.01,
			FloatFormat:    fmt.Sprintf("%%.%df", util.NumDecPlaces(0.01)),
		},
		"LTC-USD": product_info.Info{
			Platform:       "Bitstamp",
			DatabaseKey:    "Bitstamp-LTC-USD",
			ID:             "LTC-USD",
			DisplayName:    "LTC-USD",
			BaseCurrency:   "LTC",
			QuoteCurrency:  "USD",
			BaseMinSize:    0,
			BaseMaxSize:    0,
			QuoteIncrement: 0.01,
			FloatFormat:    fmt.Sprintf("%%.%df", util.NumDecPlaces(0.01)),
		},
		"XRP-USD": product_info.Info{
			Platform:       "Bitstamp",
			DatabaseKey:    "Bitstamp-XRP-USD",
			ID:             "XRP-USD",
			DisplayName:    "XRP-USD",
			BaseCurrency:   "XRP",
			QuoteCurrency:  "USD",
			BaseMinSize:    0,
			BaseMaxSize:    0,
			QuoteIncrement: 0.01,
			FloatFormat:    fmt.Sprintf("%%.%df", util.NumDecPlaces(0.01)),
		},
		"BCH-USD": product_info.Info{
			Platform:       "Bitstamp",
			DatabaseKey:    "Bitstamp-BCH-USD",
			ID:             "BCH-USD",
			DisplayName:    "BCH-USD",
			BaseCurrency:   "BCH",
			QuoteCurrency:  "USD",
			BaseMinSize:    0,
			BaseMaxSize:    0,
			QuoteIncrement: 0.01,
			FloatFormat:    fmt.Sprintf("%%.%df", util.NumDecPlaces(0.01)),
		},
	}
	// btcusd, btceur, eurusd, xrpusd, xrpeur, xrpbtc, ltcusd, ltceur, ltcbtc, ethusd, etheur, ethbtc, bchusd, bcheur, bchbtc
}

func FetchProductInfo(id string) product_info.Info {
	if info, ok := CachedInfo[id]; ok {
		return info
	}
	return product_info.Info{}
}
