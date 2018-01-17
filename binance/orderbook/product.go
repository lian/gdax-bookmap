package orderbook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
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

			info := product_info.Info{
				ID:            i["symbol"].(string),
				DisplayName:   fmt.Sprintf("%s-%s", i["baseAsset"].(string), i["quoteAsset"].(string)),
				BaseCurrency:  i["baseAsset"].(string),
				QuoteCurrency: i["quoteAsset"].(string),
			}

			if filters, ok := i["filters"].([]interface{}); ok {
				for _, f := range filters {
					fi := f.(map[string]interface{})
					if fi["filterType"].(string) == "PRICE_FILTER" {
						t, _ := strconv.ParseFloat(fi["minPrice"].(string), 64)
						info.BaseMinSize = product_info.FloatString(t)
						t, _ = strconv.ParseFloat(fi["maxPrice"].(string), 64)
						info.BaseMaxSize = product_info.FloatString(t)
						info.QuoteIncrement = info.BaseMinSize
						info.FloatFormat = fmt.Sprintf("%%.%df", NumDecPlaces(float64(info.QuoteIncrement)))
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

func NumDecPlaces(v float64) int {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	if i > -1 {
		return len(s) - i - 1
	}
	return 0
}
