package orderbook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/lian/gdax-bookmap/orderbook/product_info"
	"github.com/lian/gdax-bookmap/util"
)

var CachedInfo map[string]product_info.Info

func init() {
	FetchAllProductInfo()
}

func FetchAllProductInfo() {
	CachedInfo = map[string]product_info.Info{}

	res, err := http.Get("https://api.gdax.com/products")
	if err != nil {
		fmt.Println("InitProduct error", err)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		fmt.Println("InitProduct error", err)
	}

	var data []product_info.Info
	json.Unmarshal(body, &data)

	for _, product := range data {
		product.Platform = "GDAX"
		product.DatabaseKey = fmt.Sprintf("GDAX-%s-%s", product.BaseCurrency, product.QuoteCurrency)
		product.FloatFormat = fmt.Sprintf("%%.%df", util.NumDecPlaces(float64(product.QuoteIncrement)))
		CachedInfo[product.ID] = product
	}
}

func FetchProductInfo(id string) product_info.Info {
	if info, ok := CachedInfo[id]; ok {
		return info
	}
	return product_info.Info{}
}
