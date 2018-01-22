package websocket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/lian/gdax-bookmap/bitstamp/orderbook"
)

func (c *Client) SyncBook(book *orderbook.Book) error {
	fmt.Println("sync", book.ID)

	url := fmt.Sprintf("https://www.bitstamp.net/api/v2/order_book/%s", book.WebsocketID)
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

	if _, ok := data["timestamp"]; ok {
		book.Clear()
		seq, _ := strconv.ParseInt(data["timestamp"].(string), 10, 64)
		book.Sequence = uint64(seq)

		t := time.Now()

		if bids, ok := data["bids"].([]interface{}); ok {
			for i := len(bids) - 1; i >= 0; i-- {
				data := bids[i].([]interface{})
				price, _ := strconv.ParseFloat(data[0].(string), 64)
				size, _ := strconv.ParseFloat(data[1].(string), 64)
				book.UpdateBidLevel(t, price, size)
			}
		}

		if asks, ok := data["asks"].([]interface{}); ok {
			for i := len(asks) - 1; i >= 0; i-- {
				data := asks[i].([]interface{})
				price, _ := strconv.ParseFloat(data[0].(string), 64)
				size, _ := strconv.ParseFloat(data[1].(string), 64)
				book.UpdateAskLevel(t, price, size)
			}
		}

		if c.dbEnabled {
			batch := c.BatchWrite[book.ID]
			now := time.Now()
			fmt.Println("STORE INIT SYNC", book.ID, book.Sequence, batch.Count)
			c.WriteSync(batch, book, now)
		}
	}

	return nil
}
