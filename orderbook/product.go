package orderbook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type FloatString float64

func (t FloatString) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatFloat(float64(t), 'g', -1, 64))
}

func (t *FloatString) UnmarshalJSON(data []byte) error {
	var err error
	var s string
	var f float64
	if err = json.Unmarshal(data, &s); err != nil {
		return err
	}
	if f, err = strconv.ParseFloat(s, 64); err != nil {
		return err
	}
	*t = FloatString(f)
	return nil
}

type ProductInfo struct {
	ID             string      `json:"id"`
	DisplayName    string      `json:"display_name"`
	BaseCurrency   string      `json:"base_currency"`
	QuoteCurrency  string      `json:"quote_currency"`
	BaseMinSize    FloatString `json:"base_min_size"`
	BaseMaxSize    FloatString `json:"base_max_size"`
	QuoteIncrement FloatString `json:"quote_increment"`
	FloatFormat    string
}

func (p ProductInfo) FormatFloat(v float64) string {
	return fmt.Sprintf(p.FloatFormat, v)
}

func FetchProductInfo(id string) ProductInfo {
	var product ProductInfo
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

	var data []ProductInfo
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
