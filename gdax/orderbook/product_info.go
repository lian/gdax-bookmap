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

func FetchProductInfo(id string) product_info.Info {
	var product product_info.Info
	res, err := http.Get("https://api.gdax.com/products")
	if err != nil {
		fmt.Println("InitProduct error", err)
		return product
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		fmt.Println("InitProduct error", err)
		return product
	}

	var data []product_info.Info
	json.Unmarshal(body, &data)
	for _, product = range data {
		if product.ID == id {
			product.FloatFormat = fmt.Sprintf("%%.%df", NumDecPlaces(float64(product.QuoteIncrement)))
			return product
		}
	}
	return product
}

func NumDecPlaces(v float64) int {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	if i > -1 {
		return len(s) - i - 1
	}
	return 0
}
