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

	res, err := http.Get("https://api.bitfinex.com/v1/symbols_details")
	if err != nil {
		fmt.Println("InitProduct error", err)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		fmt.Println("InitProduct error", err)
	}

	var data []interface{}
	json.Unmarshal(body, &data)

	for _, d := range data {
		i := d.(map[string]interface{})

		pair := i["pair"].(string)

		if len(pair) != 6 {
			continue
		}

		pair = strings.ToUpper(pair)
		base := strings.ToUpper(pair[0:3])
		quote := strings.ToUpper(pair[3:])

		info := product_info.Info{
			ID:            pair,
			DisplayName:   fmt.Sprintf("%s-%s", base, quote),
			BaseCurrency:  base,
			QuoteCurrency: quote,
			Platform:      "Bitfinex",
			DatabaseKey:   fmt.Sprintf("Bitfinex-%s-%s", base, quote),
		}

		t, _ := strconv.ParseFloat(i["minimum_order_size"].(string), 64)
		info.BaseMinSize = product_info.FloatString(t)
		t, _ = strconv.ParseFloat(i["maximum_order_size"].(string), 64)
		info.BaseMaxSize = product_info.FloatString(t)

		info.QuoteIncrement = info.BaseMinSize
		//info.FloatFormat = fmt.Sprintf("%%.%df", util.NumDecPlaces(float64(info.QuoteIncrement)))
		if info.QuoteCurrency == "USD" || info.QuoteCurrency == "EUR" {
			info.FloatFormat = "%.2f"
		} else {
			info.FloatFormat = "%.5f"
		}

		CachedInfo[info.DisplayName] = info
	}
}

func FetchProductInfo(id string) product_info.Info {
	if info, ok := CachedInfo[id]; ok {
		return info
	}
	return product_info.Info{}
}
