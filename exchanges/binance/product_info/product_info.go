package product_info

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
	"github.com/lian/gdax-bookmap/util"
)

var CachedInfo map[string]product_info.Info

func init() {
	FetchAllProductInfo()
}

func FetchAllProductInfo() {
	CachedInfo = map[string]product_info.Info{}

	res, err := http.Get("https://api.binance.com/api/v1/exchangeInfo")
	if err != nil {
		fmt.Println("InitProduct error", err)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		fmt.Println("InitProduct error", err)
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	if symbols, ok := data["symbols"].([]interface{}); ok {
		for _, p := range symbols {
			i := p.(map[string]interface{})

			baseAsset := i["baseAsset"].(string)
			if baseAsset == "BCC" {
				baseAsset = "BCH"
			}

			info := product_info.Info{
				ID:            i["symbol"].(string),
				DisplayName:   fmt.Sprintf("%s-%s", baseAsset, i["quoteAsset"].(string)),
				BaseCurrency:  baseAsset,
				QuoteCurrency: i["quoteAsset"].(string),
				Platform:      "Binance",
				DatabaseKey:   fmt.Sprintf("Binance-%s-%s", baseAsset, i["quoteAsset"].(string)),
			}

			if filters, ok := i["filters"].([]interface{}); ok {
				for _, f := range filters {
					fi := f.(map[string]interface{})
					if fi["filterType"].(string) == "PRICE_FILTER" {
						t, _ := strconv.ParseFloat(fi["minPrice"].(string), 64)
						info.BaseMinSize = t
						t, _ = strconv.ParseFloat(fi["maxPrice"].(string), 64)
						info.BaseMaxSize = t
						info.QuoteIncrement = info.BaseMinSize
						info.FloatFormat = fmt.Sprintf("%%.%df", util.NumDecPlaces(float64(info.QuoteIncrement)))
						CachedInfo[info.DisplayName] = info
						break
					}
				}
			}

		}
	}
}

func FetchProductInfo(id string) product_info.Info {
	if info, ok := CachedInfo[id]; ok {
		return info
	}
	return product_info.Info{}
}
