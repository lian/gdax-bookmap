package product_info

import "fmt"

type Info struct {
	DatabaseKey    string
	Platform       string
	ID             string  `json:"id"`
	DisplayName    string  `json:"display_name"`
	BaseCurrency   string  `json:"base_currency"`
	QuoteCurrency  string  `json:"quote_currency"`
	BaseMinSize    float64 `json:"base_min_size,string"`
	BaseMaxSize    float64 `json:"base_max_size,string"`
	QuoteIncrement float64 `json:"quote_increment,string"`
	FloatFormat    string
}

func (i Info) FormatFloat(v float64) string {
	return fmt.Sprintf(i.FloatFormat, v)
}
