package product_info

import (
	"encoding/json"
	"fmt"
	"strconv"
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

type Info struct {
	ID             string      `json:"id"`
	DisplayName    string      `json:"display_name"`
	BaseCurrency   string      `json:"base_currency"`
	QuoteCurrency  string      `json:"quote_currency"`
	BaseMinSize    FloatString `json:"base_min_size"`
	BaseMaxSize    FloatString `json:"base_max_size"`
	QuoteIncrement FloatString `json:"quote_increment"`
	FloatFormat    string
}

func (i Info) FormatFloat(v float64) string {
	return fmt.Sprintf(i.FloatFormat, v)
}
