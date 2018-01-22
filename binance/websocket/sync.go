package websocket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lian/gdax-bookmap/binance/orderbook"
)

func (c *Client) SyncBook(book *orderbook.Book) error {
	fmt.Println("sync", book.ID)

	url := fmt.Sprintf("https://www.binance.com/api/v1/depth?symbol=%s&limit=1000", strings.ToUpper(book.ProductInfo.ID))
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	if seq, ok := data["lastUpdateId"]; ok {
		book.Clear()
		book.Sequence = uint64(seq.(float64))

		t := time.Now()

		if bids, ok := data["bids"].([]interface{}); ok {
			for i := len(bids) - 1; i >= 0; i-- {
				data := bids[i].([]interface{})
				price, _ := strconv.ParseFloat(data[0].(string), 64)
				quantity, _ := strconv.ParseFloat(data[1].(string), 64)
				book.UpdateBidLevel(t, price, quantity)
			}
		}

		if asks, ok := data["asks"].([]interface{}); ok {
			for i := len(asks) - 1; i >= 0; i-- {
				data := asks[i].([]interface{})
				price, _ := strconv.ParseFloat(data[0].(string), 64)
				quantity, _ := strconv.ParseFloat(data[1].(string), 64)
				book.UpdateAskLevel(t, price, quantity)
			}
		}

		if c.dbEnabled {
			now := time.Now()
			fmt.Println("STORE INIT SYNC", book.ID)
			c.WriteDB(now, book, PackSync(book))
		}
	}

	return nil
}
